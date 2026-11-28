# DrukLeague — Architecture

Version 28 Nov 2026. This describes the system as implemented in this repository; the dependencies below are taken from `deploy/docker-compose.yml` and the service code, not from the plan. The diagrams use Mermaid and render on GitHub.

## 1. System overview

```mermaid
flowchart TB
  subgraph Client
    browser["Web app (HTML/CSS/JS)<br/>landing page + role workspace"]
  end
  subgraph Edge["Edge (nginx)"]
    nginx["nginx: static files, CSP and security headers,<br/>rate limits (login, sign-up, public reads)"]
  end
  subgraph Access["Access layer"]
    bff["BFF (Go, REST → gRPC)<br/>session cookies, public routes, sign-up"]
    api["API gateway / monolith (Go)<br/>legacy REST, admin corrections"]
  end
  subgraph Services["20 gRPC microservices (Go, hexagonal)"]
    direction LR
    auth[Auth] --- user[User]
    team[Team] --- player[Player] --- tournament[Tournament] --- registration[Registration]
    match[Match] --- score[Score] --- fixture[Fixture] --- standings[Standings]
    venue[Venue] --- official[Official] --- dzongkhag[Dzongkhag]
    notification[Notification] --- media[Media] --- report[Report] --- search[Search]
    analytics[Analytics] --- audit[Audit Log] --- config[Configuration]
  end
  subgraph Serverless["Serverless (OpenFaaS of-watchdog)"]
    fnreport[["match-report"]]
    fncsv[["standings-csv"]]
    connector["Function connector"]
  end
  js[("NATS JetStream<br/>DRUKLEAGUE_EVENTS + DLQ")]
  pg[("PostgreSQL 17<br/>one database per service")]
  obs["Prometheus + Grafana"]

  browser --> nginx
  nginx -->|/api/*| bff
  nginx -->|/api/reports, /api/media| report & media
  nginx -->|/fn/standings-csv| fncsv
  bff -->|gRPC, caller's JWT| Services
  api --> js
  Services -->|own schema only| pg
  Services <-->|events| js
  js --> connector --> fnreport -->|public reads| bff
  fncsv -->|public reads| bff
  obs -.->|scrape /metrics| Services
```

- **Synchronous calls** use gRPC with versioned Protocol Buffer contracts (`proto/`, checked by Buf lint/breaking in CI). Every call carries a deadline, retries safe reads with backoff, and passes through a per-target circuit breaker (`packages/servicekit`).
- **Authentication:** the Auth service issues HS256 JWTs (30-minute access tokens with rotating refresh tokens). The BFF stores them in HttpOnly, SameSite=Strict cookies and forwards the caller's token, so **every service checks the caller's role itself**. Public routes and viewer sign-up use a short-lived service token instead and never forward browser credentials.
- **Data ownership:** each service owns one PostgreSQL database (for example `score_service` or `registration_service`). Nothing reads another service's database; cross-service data travels over gRPC or events.

## 2. Service responsibilities and dependencies

| Service | Owns | Calls (gRPC) | Events |
|---|---|---|---|
| Auth | sessions, refresh tokens | User | — |
| User | accounts, roles, password hashes | — | — |
| Team | teams, managers | — | publishes search documents |
| Player | squads | Team | publishes search documents |
| Tournament | tournaments | — | publishes search documents |
| Registration | tournament entries (pending/approved/rejected) | Team, Tournament | — |
| Fixture | schedules, knockout plans | Tournament, Venue, Registration, Match, Standings | publishes `FixturesGenerated` |
| Match | fixtures, official assignment | — | consumes `MatchFinalized` |
| Score | live and final scores (immutable once final) | Match | publishes `MatchFinalized` via a transactional outbox |
| Standings | league tables | Team | consumes `MatchFinalized` (idempotent by `event_id`) |
| Venue | grounds per dzongkhag | Dzongkhag | — |
| Official | assignments | Match | — |
| Dzongkhag | the 20 dzongkhags | — | — |
| Notification | notification records | Team | consumes `MatchFinalized` |
| Media | uploads (HTTP + gRPC) | — | — |
| Report | generated reports (HTTP) | — | — |
| Search | search index | — | consumes search documents |
| Analytics | dashboard facts | Tournament, Team, Match, Player, Registration | consumes `MatchFinalized` |
| Audit Log | admin action history | — | consumes `drukleague.admin.>` |
| Configuration | allow-listed runtime settings | — | — |

The **BFF** calls Auth, User, Tournament, Team, Registration, Player, Venue, Match, Score, Fixture, Standings and Official.

## 3. Event flow

```mermaid
flowchart LR
  score[Score] -->|outbox| e1(["drukleague.match.finalized"])
  fixture[Fixture] -->|outbox| e2(["drukleague.fixtures.generated"])
  api[API gateway admin actions] -->|audit outbox| e3(["drukleague.admin.>"])
  team[Team, Player, Tournament] --> e4(["drukleague.search.*"])
  e1 --> match[Match] & standings[Standings] & notification[Notification] & analytics[Analytics] & connector[Function connector]
  e3 --> audit[Audit Log]
  e4 --> search[Search]
  connector --> fn[[match-report function]] --> r(["drukleague.functions.match-report"])
  e1 & e2 & e3 & e4 -.->|after 5 failed deliveries| dlq[("DRUKLEAGUE_DLQ")]
```

Delivery is at-least-once: messages carry stable ids (`Nats-Msg-Id`) for publisher de-duplication, consumers are durable and idempotent, redelivery backs off, and poison messages move to the dead-letter stream (30-day retention) where `tools/event-admin` can inspect and replay them.

## 4. Key workflow: finalising a result

```mermaid
sequenceDiagram
  actor O as Match official
  participant B as BFF
  participant S as Score
  participant M as Match
  participant J as JetStream
  participant St as Standings
  participant C as Function connector
  participant F as match-report
  O->>B: POST /api/matches/{id}/finalize {home, away}
  B->>S: UpdateLiveScore + FinalizeMatch (official's JWT)
  S->>M: GetMatch (is the caller the assigned official?)
  S->>S: one transaction: final score + outbox row
  S-->>B: final score (later edits return 409)
  S->>J: publish MatchFinalized (outbox relay)
  J->>St: deliver → update table once per event_id
  J->>C: deliver
  C->>F: POST event
  F->>B: GET /api/public/matches, /standings
  F-->>C: announcement JSON
  C->>J: publish drukleague.functions.match-report
```

## 5. Deployment view

```mermaid
flowchart LR
  subgraph Host["Docker host (Compose project 'deploy')"]
    web["web: nginx 1.27<br/>127.0.0.1:4173"]
    bffc["bff<br/>127.0.0.1:8080"]
    svc["20 service containers<br/>distroless, non-root"]
    fns["fn-match-report, fn-standings-csv,<br/>function-connector"]
    infra["postgres 17, nats 2 (JetStream),<br/>prometheus, grafana"]
  end
  user((Browser)) --> web --> bffc --> svc --> infra
  web --> fns --> bffc
```

- Every image builds from `services/service-builder` (Go 1.26) into `gcr.io/distroless/static-debian12:nonroot`. CI builds all images and fails on CRITICAL Trivy findings.
- Ports are bound to `127.0.0.1` only. Secrets come from `deploy/.env`, which is git-ignored.
- The deployment is local (Docker Compose), as scoped in the proposal ("production-scale cloud deployment" is out of scope). The functions are also deployable to OpenFaaS/faasd with `functions/stack.yml`.

## 6. Quality attributes and where they are handled

| Attribute | Mechanism | Location |
|---|---|---|
| Security | JWT + role checks in every service, HttpOnly cookies, CSP, rate limits, sign-up fixed to Viewer, formula-safe CSV | `servicekit/auth.go`, each `internal/app`, `deploy/nginx*.conf`, `services/bff` |
| Integrity | the assigned official only; final scores immutable; outbox in the same transaction | `services/score`, `services/match` |
| Resilience | deadlines, retry with backoff, circuit breaker, durable consumers, DLQ, graceful degradation in BFF lists and functions | `packages/servicekit`, `packages/eventkit` |
| Observability | request ids end to end, Prometheus metrics, Grafana dashboards | `servicekit/metrics.go`, `monitoring/` |
| Maintainability | hexagonal layout per service (domain, app, ports, adapters) | `services/*/internal` |
