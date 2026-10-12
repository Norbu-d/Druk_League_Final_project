# DrukLeague — Project Progress and Handoff

**Last updated:** 2026-10-08
**Project root:** `D:\druk-league`
**Run state:** Docker verified the stack earlier on 2026-10-08: all 20 service containers were Up, PostgreSQL and NATS healthy, and API `/healthz` and web returned HTTP 200. I have since implemented domain RPCs in source for User, Team, Player, Registration, and Tournament, and those five modules compile locally; Dzongkhag Get/List was already implemented. The service Dockerfiles/build script now compile each module into its own image. The first full per-service image rebuild overloaded Docker BuildKit and returned an HTTP 500; I cancelled it, and Docker CLI has not yet been verified responsive again. New source handlers are not confirmed in the running containers. The PostgreSQL 16 volume `drukleague-postgres-data` remains preserved; Compose uses the separate PostgreSQL 17 volume `drukleague-postgres17-data`. Next resume by recovering Docker and building changed images at lower parallelism, then continue service implementation.

> **Read this first.** The code is a working **MVP vertical slice** (one Go API + plain HTML/CSS/JS frontend). The proposal and the presentation describe a much larger system (20 microservices + gateway + BFF + serverless functions). Section 2 lists exactly where they differ. Do not report the MVP as the finished architecture in reports or demos.

---

## 1. Project overview and goal

DrukLeague is a tournament operations platform for Bhutan's inter-dzongkhag sports competitions. It replaces coordination by phone calls, paper score sheets and messaging apps with one place to create tournaments, register teams, generate fixtures, submit live scores, finalise results and publish standings.

**Course context:** one project covering three modules, group of four (Tandin Wangchuk, Norbu Dendup, Sonam Choki, Namgay Lhamo).

| Module | Tutor | What it assesses here |
|---|---|---|
| WEB303 Microservices & Serverless Applications | Mr. Pema Namgay | 20 services, gRPC, NATS, OpenFaaS functions |
| SWE302 Software Testing & QA | Mr. Pema Namgay | 80% unit-test coverage per service, CI gates, security/performance tests |
| SWE303 Software Project Management | Mr. Nima Yoezer | Scrum (8 two-week sprints, 16 weeks), risks, test plan, traceability |

**Roles (target):** tournament admin (called Super Admin in the code), match official, team user, viewer (no login). **There is no Dzongkhag Admin role** (dropped 7 Oct 2026): dzongkhag is reference data (a Dzongkhag service holding the 20 dzongkhags) and a field on teams and tournaments, used for filtering. One responsive web app with role-based screens; authorisation enforced server-side at the gateway **and** inside each service, not only in the UI.

**Core rule:** live scores are mutable; a result becomes final only when the *assigned* match official finalises it. Final results are immutable and are the only source for standings updates and downstream events (`MatchFinalized`).

**Budget:** Nu. 0 — free/open-source tools, run locally with Docker Compose. Out of scope: production cloud deployment, paid hosting, real SMS/payment gateways, native mobile app, production-level scaling.

**Proposal versions (keep straight):**
- `DrukLeague_Project_Proposal.docx` (6 Oct 2026) — the current proposal: 20 core microservices + API Gateway + BFF + OpenFaaS functions, Go, web only. The presentation matches this one.
- `DrukLeague_SWE303_Project_ProposalV1.docx` — smaller draft (about 6 services + 2 serverless functions). The repo's current plan is closer to this one.
- An earlier 22-service TypeScript/NestJS plan with a mobile app was dropped.

---

## 2. Status vs the proposal (gap analysis)

| Proposal / presentation says | Code today |
|---|---|
| 20 independently deployable microservices (core deliverable, not future work) | Legacy Go API still contains business logic; 20 service modules and contracts now exist as deployable shells, with Dzongkhag get/list implemented separately |
| API Gateway + BFF in front of services | The monolith *is* the "gateway"; no BFF |
| gRPC + Protocol Buffers between services | 20 domain service contracts plus event contract; Buf lint/build and Go/gRPC generation pass; Dzongkhag, User, Team, Player, Registration, and Tournament handlers are implemented in source; module-specific images still need rebuilding |
| Each service owns its PostgreSQL data | One shared database, one startup migration |
| NATS JetStream event flows; prototype "demonstrated NATS" | Outbox publisher code written; **never connected to a running NATS server; no consumers** |
| OpenFaaS serverless functions | **None** |
| OpenTelemetry tracing, circuit breaker, fault isolation | Not started |
| 80% unit-test coverage per service, CI quality gates | **Zero automated tests, no CI** |
| Hexagonal architecture (Cockburn, 2005) cited in the literature review | Cited only: the proposal never says services will use it, and `main.go` mixes handlers, rules and SQL |
| Fault isolation tested in Sprint 7; reliable event publishing | Outbox, stable message IDs and `/healthz` exist; **no timeouts, retries, circuit breaker, idempotent consumers or dead-letter handling** |
| Deck/tech slide: JWT + **bcrypt** | Code uses **PBKDF2-HMAC-SHA256** (and hand-rolled HMAC JWT) |
| Single Tournament Admin role (Dzongkhag Admin dropped) | Removed from the MVP API; districts are seeded reference data and team/tournament fields |
| Team users register teams and track them | Team-user role remains unimplemented; current code roles are Super Admin, Match Official, Viewer |
| Docker Compose runs the stack | Compose config validates; all 20 service images built and all 20 service containers are running. PostgreSQL and NATS are healthy; API `/healthz` and web return HTTP 200 |

### Status of the 20 core services
Monolith means business logic still lives in the old API. All 20 modules have separate migrations, Dockerfiles, and protobuf contracts. Earlier generic images for all 20 services ran in Compose and API/web checks passed. Service Dockerfiles now compile each module into its own image. Dzongkhag, User, Team, Player, Registration, and Tournament have domain handlers in source and pass local module builds. The other 14 modules still expose health shells. The new per-module image rebuild hit an HTTP 500 from Docker BuildKit and was cancelled; deploy status must be rechecked after Docker recovers.

| Owner (per proposal) | Service | Status |
|---|---|---|
| Tandin | Auth | In monolith; separate module/contract shell; domain RPC pending |
| Tandin | User | In monolith; Create/Get/List RPCs with password hashing and pagination implemented in source; local build passes; container rebuild pending |
| Tandin | Team | In monolith; Create/Get/List RPCs with filtering and pagination implemented in source; local build passes; container rebuild pending |
| Tandin | Player | Create/Get/List/Update RPCs implemented in source; local build passes; container rebuild pending |
| Tandin | Tournament | In monolith; Create/Get/List RPCs implemented in source; local build passes; container rebuild pending |
| Tandin | Registration | Register/Review/List RPCs implemented in source; local build passes; container rebuild pending |
| Norbu | Match | In monolith; separate module/contract shell; domain RPC pending |
| Norbu | Score | In monolith; separate module/contract shell; domain RPC pending |
| Norbu | Fixture | Round-robin generation remains in monolith; separate module/contract shell |
| Norbu | Standings | Updated on finalise in monolith; separate module/contract shell |
| Norbu | Venue | Venue field/clash check remains in monolith; separate module/contract shell |
| Sonam | Official | Assignment endpoint remains in monolith; separate module/contract shell |
| Sonam | Dzongkhag | Separate gRPC List/Get implemented with PostgreSQL seed of all 20 districts |
| Sonam | Notification, Audit Log, Media | Separate module/contract shells; domain RPC pending |
| Namgay | Report, Search, Analytics, Configuration | Separate module/contract shells; domain RPC pending |

---

## 3. Tech stack

### Implemented in the MVP
- **Frontend:** plain HTML, CSS, JavaScript; no build step or package manager.
- **Backend:** Go HTTP API in `services/api-gateway`.
- **Database:** PostgreSQL via `pgx/v5` / `database/sql`; one shared MVP database, one startup SQL migration.
- **Auth:** custom HMAC-SHA256 JWT and PBKDF2-HMAC-SHA256 password hashing.
- **Events:** NATS JetStream client + transactional outbox publisher (code only).
- **Contracts:** 20 domain service contracts plus MatchFinalized; Buf v2 lint/build pass and Go/gRPC stubs generated under packages/contracts/gen.
- **Local deployment:** Compose defines PostgreSQL 17, NATS, Nginx, API and 20 services. deploy/build-services.ps1 builds a shared runtime and service images; full stack health is currently unverified.

### Planned (per proposal) but not implemented
Domain gRPC methods in 19 service shells; API Gateway + BFF; per-service databases; JetStream consumers; OpenFaaS/faasd functions; OpenTelemetry; per-service tests and CI;

---

## 4. Completed work and file guide

### Frontend and brand
- `index.html` — app shell, sidebar navigation, top bar, content/modal roots; DrukLeague logo.
- `styles.css` — responsive visual system, readability overrides, logo placement.
- `app.js` — prototype screens: dashboard, tournament/team forms, fixtures, score modal, standings, officials/venues/reports views, filters, CSV export. Keeps illustrative sample data when signed out.
- `api.js` — sign-in and API client; loads tournaments, teams, matches, standings; wires tournament/team creation, fixture generation, live/final score submission.
- `assets/drukleague-logo.png` — supplied logo.
- `web-preview/` — restricted copy of the frontend used for the localhost preview. **Edit the root files as the source of truth.**

### Backend, schema, contracts
- `services/api-gateway/go.mod`, `go.sum` — module + checksums (`pgx/v5`, `nats.go`).
- `services/api-gateway/cmd/api/main.go` — HTTP server, startup migration/bootstrap, JWT middleware, role checks, tournament/team/match/standings handlers, fixture generation, live scoring, transactional finalisation, standings updates.
- `services/api-gateway/cmd/api/admin_matches.go` — super-admin account listing/creation; match-official assignment; match creation.
- `services/api-gateway/cmd/api/events.go` — creates the JetStream stream; drains the outbox to `drukleague.match.finalized` with stable message IDs.
- `services/api-gateway/migrations/001_init.sql` — users, tournaments, teams, matches, standings, event outbox, indexes, and a trigger that blocks edits to finalised result fields.
- `services/api-gateway/Dockerfile` — multi-stage API image.
- proto/ now contains contracts for all 20 services plus the MatchFinalized event.
- packages/contracts/ contains generated Go/protobuf code; packages/servicekit/ contains the shared PostgreSQL migration and gRPC-health runtime; services/_template/ is the hexagonal service example.

### Deployment and configuration
- `deploy/docker-compose.yml` — PostgreSQL, NATS JetStream, API, web.
- `deploy/nginx.conf` — static UI + proxy for `/api` and `/healthz`.
- `deploy/.env.example` — template. `deploy/.env` exists locally and is **private**: never print, copy or commit it.
- `README.md` — setup, API routes, functionality, known limits.

### Course documents (in `docs/`)
- `architecture-decision-mvp.md` — why the slice is consolidated; score-correction and fixture-clash assumptions.
- `stakeholder-analysis.md`, `wbs.md`, `risk-register.md`, `change-control.md` — SWE303.
- `swe302-test-plan.md` — planned test scenarios (**not executed**).
- `defect-log.md`, `traceability-matrix.md` — open gaps; requirement → code → verification mapping.

### Presentation and proposal artifacts
- `DrukLeague_Redesigned_v2.pptx` — 19-slide deck on the college template (Arial body, pastel icon-card style, icons taken from the supplied design images). Includes event-driven architecture, three-modules and references slides. Speaker notes on every slide with speaker tag and timing (~16:45 total; Tandin 1–4, Norbu 5–9, Sonam 10–13, Namgay 14–19).
- `DrukLeague_Project_Proposal_v2.docx` (Dzongkhag Admin wording removed; supersedes `DrukLeague_Project_Proposal.docx`), `DrukLeague_Project_Proposal.docx`, `DrukLeague_SWE303_Project_ProposalV1.docx`, `Proposal Template.docx` — proposal files.
- Proposal review (graded about 74/100 before fixes). Main points: references section empty; problem lacks evidence; 20 services in 16 weeks is ambitious; sprint 4 clash detection depends on Venue in Sprint 5; testing back-loaded; thin SWE303 content.

### Implemented behaviour
- Auth: `Authorization: Bearer <token>`; `POST /api/auth/login`.
- Super Admin creates match officials and viewers. There is no Dzongkhag Admin role; team and tournament writes are admin-only, while dzongkhags are reference data.
- Match creation rejects same-kickoff team/venue clashes; round-robin generation is guarded by a per-tournament advisory lock and won't duplicate.
- Only the assigned match official can update or finalise a score.
- Finalisation writes the result, both standings rows and the outbox event in one transaction; a DB trigger rejects later edits.

| Route | Purpose |
|---|---|
| `/api/users` | list/create accounts (Super Admin) |
| `/api/tournaments` | create/read/list |
| `/api/teams` | register/list |
| `/api/matches` | create/list; `/{id}/official`, `/{id}/live`, `/{id}/finalize` |
| `/api/tournaments/{id}/fixtures` | generate round-robin fixtures |
| `/api/standings?tournament_id=...` | read standings |
| `/healthz` | health |

---

## 5. Key code and implementation notes

The complete source lives in the files above (it is not reproduced here). Key flow in `services/api-gateway/cmd/api/main.go`:

1. Startup connects to PostgreSQL, applies `001_init.sql`, bootstraps a super-admin if none exists.
2. Login verifies the PBKDF2 hash and returns an HMAC-signed JWT; middleware verifies signature and expiry for `/api/*`.
3. Tournament/team handlers validate required fields; all tournament writes are limited to the tournament admin and dzongkhag is reference data.
4. Fixture generation runs in a transaction under an advisory lock; explicit match creation rejects same-time clashes.
5. The assigned official updates the live score, then finalises once; finalisation atomically writes immutable scores, standings and an outbox row.
6. `events.go` polls the outbox and publishes committed events to JetStream (never exercised against a live NATS server).

**Checks that passed:** Go API build; Go builds for all 20 base modules; updated User, Team, Player, Registration, and Tournament modules; Dzongkhag service build; Buf 1.73.0 lint/build/generate; JavaScript syntax checks; Compose config validation. The prior generic service image build/runtime were verified; the newer per-module image rebuild is incomplete because Docker BuildKit became unresponsive. No automated test suite has been run.

**Structure note:** handlers, business rules and SQL currently live together in `main.go`. The service split should move to ports and adapters (section 5a).

**Security notes:** custom JWT/PBKDF2 code is acceptable for a prototype but should be replaced by a vetted library (e.g. `golang-jwt`) and bcrypt/argon2 to match the slides; add server-side role + ownership checks in every service; clients must never be able to set their own role.

---

## 5a. Target service design: hexagonal architecture and resilience

### Hexagonal (ports and adapters), applied per service
Cockburn (2005) is cited in the proposal's literature review; this is how each service should use it.

```text
services/match/
├── internal/domain/        # Match, Score, rules (immutable once finalised)
├── internal/app/           # use cases: SubmitScore, FinaliseMatch
├── internal/ports/         # interfaces: MatchRepo, EventPublisher, TeamLookup
├── internal/adapters/
│   ├── grpc/               # inbound: gRPC handlers
│   ├── postgres/           # outbound: repository + outbox table
│   ├── nats/               # outbound: JetStream publisher
│   └── teamclient/         # outbound: gRPC client to the Team service
└── cmd/match/main.go       # wiring only
```

- The core (domain + app) imports no database, gRPC or NATS packages. Adapters depend on the core, never the reverse.
- Ports are interfaces owned by the core. Inbound adapters call use cases; outbound adapters implement ports.
- Unit tests use fakes or GoMock for the ports (no database or NATS needed), which is what the 80% coverage target relies on. Integration tests (Testcontainers) cover the adapters.
- Hexagonal does not make a system resilient by itself; it tells you where resilience goes: in the **outbound adapters**, so domain code does not change.

### Resilience patterns
| Pattern | Where it lives | Status |
|---|---|---|
| Transactional outbox (result + standings + event in one transaction) | Match's postgres adapter | Implemented in the MVP; carry into the Match service |
| Stable message IDs (publisher de-duplication) | NATS publisher adapter | In the MVP publisher |
| Durable JetStream, redelivery limit, dead-letter stream | NATS consumer adapters | Planned |
| Idempotent consumers (skip already-processed event IDs) | app layer | Planned |
| gRPC deadlines (timeouts) | outbound client adapters | Planned |
| Retry with backoff, safe reads only | outbound client adapters | Planned |
| Circuit breaker (Match → Team, gateway/BFF → services) | outbound client adapters | Planned; needed for the Sprint 7 demo |
| Database per service | each service | Planned (shared DB in the MVP) |
| Health and readiness endpoints | each service | `/healthz` in the MVP only |
| Graceful degradation (reads keep working if Notification is down) | app layer | Planned |
| OpenTelemetry tracing | all services | Planned |
| Rate limiting | gateway | Not planned |

**Fault-isolation scenarios to test (Sprint 7):** stop Team, Match still accepts scores (breaker opens); stop NATS, outbox rows accumulate and drain on recovery; stop Notification, finalisation still succeeds; deliver a duplicate event, it is processed once; a poison message ends up in the dead-letter stream.

---

## 6. Open decisions (resolve first)

1. **Scope:** Decided 7 Oct 2026: build toward all 20 services as required by the proposal.
2. **Password hashing:** switch code to bcrypt, or update the deck/proposal to say PBKDF2.
3. **Team-user role:** add it (needed for team registration and "track my team"), and align role names across code, proposal and deck.
4. **OpenFaaS vs faasd:** OpenFaaS normally needs Kubernetes; faasd runs on one machine without it. Pick one and spike it early.
5. **Risk R1:** the deck rates "20 services in one semester" High/High; the proposal says Medium. Make them match.
6. **Finalised-result correction:** admin correction with a reason + audit entry (user stories) vs append-only correction events (architecture note). Choose one.
7. **Sprint 4/5 dependency:** fixture clash detection needs Venue, which is planned for Sprint 5. Reorder or stub Venue earlier.
8. **Dzongkhag Admin dropped (decided 7 Oct 2026):** one Tournament Admin role; the Dzongkhag service stays as reference data so the 20-service count holds. Proposal (`DrukLeague_Project_Proposal_v2.docx`) and deck notes already updated; API, migration, README, risk register, test plan, traceability matrix, and preview forms updated 7 Oct 2026; runtime tests remain pending.
9. **Claims to verify before presenting:** that NATS JetStream has actually run end to end.
10. **Hexagonal + resilience wording:** the proposal cites hexagonal architecture but never says it will be applied, and names no resilience mechanism beyond the outbox and fault-isolation tests. Add one methodology sentence (ports and adapters per service) and a resilience objective, then update the deck's architecture and testing slides, the risk table and the speaker notes (not done yet).

---

## 7. Remaining work (priority order)

**Completed 7 Oct 2026:** Removed Dzongkhag Admin and district-scoped authorization; seeded 20 districts; added tournament dzongkhag; updated frontend/docs. API build and JS syntax pass. Current service work: contract lint/build/codegen pass, 20 Go shells compile, Dzongkhag Get/List builds. Docker runtime is unverified after the engine stopped responding.

1. ~~Remove the Dzongkhag Admin role and update dzongkhag reference data/docs.~~ Completed 7 Oct 2026; existing tournaments retain a null dzongkhag until the owner assigns values; legacy Dzongkhag Admin accounts are demoted to Viewer by the startup migration.
2. **Get the full stack running.** PostgreSQL 16 volume verified and preserved; Compose uses drukleague-postgres17-data. MVP stack first returned HTTP 200, but Docker became unresponsive during the all-service build. Restart/recheck Docker, run deploy/build-services.ps1, start Compose and verify health.
3. **Buf:** Moved deps to workspace scope in buf.yaml; Buf 1.73.0 lint, build and Go/gRPC generation pass. Breaking check needs a valid baseline revision.
4. **Service template + split:** Shared servicekit and hexagonal template exist; all 20 modules have their own database name, migration, Dockerfile and health shell. Dzongkhag Get/List is implemented. Remaining: implement 19 domain APIs, extract monolith logic, add gateway/BFF clients, service-owned credentials, tests, then build/run all images.
5. **Events and serverless:** run NATS; consumers for notification, idempotent fixture generation, audit log and standings, all idempotent (dedupe on event ID) with limited redelivery and a dead-letter stream; OpenFaaS/faasd functions (none exist yet).
6. **Frontend:** replace sample data with API data (dashboard metrics, all-tournament standings, officials list/assignment, venues, reports); add team-user and public viewer views; loading/error/empty states.
7. **Testing (SWE302):** implement `docs/swe302-test-plan.md` (role isolation, clash handling, one-time finalisation, standings atomicity, outbox recovery); GitHub Actions with lint + tests + 80% coverage gate; gosec, govulncheck, Trivy, ZAP; k6; Playwright; fault-isolation and circuit-breaker demo; resilience tests from section 5a (breaker opens after repeated failures, retries only on safe reads, duplicate event processed once, poison message reaches the dead-letter stream).
8. **Observability:** OpenTelemetry traces across gateway and services.
9. **SWE303 evidence:** sprint plans, reviews and retrospectives; update defect log and traceability matrix with executed results; RACI/estimation/communication plan; final report and demo script.
10. **Hygiene:** `.env` in `.gitignore`; rotate demo/bootstrap credentials before sharing; bind DB/API ports to loopback for local runs; keep proposal, deck and README consistent after each change.

### Suggested sprint mapping (from the proposal's 8 sprints)
S1 foundation/CI/contracts · S2 identity + gateway + BFF · S3 teams/tournaments/registration · S4 competition (match → standings) · S5 venues/officials/notifications · S6 supporting services · S7 integration, events, serverless, observability, security/performance tests · S8 QA, coverage, defects, demo.

---

## 8. User stories (condensed, for traceability)

- **Team user:** create account/log in; register team for a tournament; add/edit players before the deadline; see fixtures with date/time/venue; get notified of changes/results; see own results and league position.
- **Viewer (no login):** browse tournaments, fixtures, live scores; view standings; search by team/player/dzongkhag; see past results.
- **Match official:** see assigned matches; record live score and submit final result; cannot change it after finalising.
- **Tournament admin:** create tournament (sport, format, dates, scoring rules); open/close registration and approve teams; generate/adjust fixtures with clash flags; manage venues and assign officials; correct a finalised result with a reason (audited); assign roles; view reports.
- **By dzongkhag (all roles):** teams and tournaments carry a dzongkhag; fixtures, results and standings can be filtered by dzongkhag. No separate dzongkhag-admin role.

Key acceptance criteria: team registration is *pending* until an admin approves and the team user is notified; an unassigned official's score submission returns 403; a team user calling an admin endpoint gets 403; a finalised match publishes `MatchFinalized` once, and standings, notification and audit react independently.

---

## 9. Continue in a new Claude chat

Open the `D:\druk-league` folder in Claude and send:

> DrukLeague is targeting all 20 services (confirmed by the project owner on 7 Oct 2026). Read progress.md and README.md; never expose deploy/.env. All 20 service images are built and their containers are running; PostgreSQL and NATS are healthy, and API `/healthz` plus the web frontend return HTTP 200. The PostgreSQL 16 volume `drukleague-postgres-data` is preserved; Compose uses the separate PostgreSQL 17 volume `drukleague-postgres17-data`. Only Dzongkhag List/Get currently has domain handlers; implement the remaining 19 services, then build gateway/BFF integrations. Do not claim unrun tests as passing.

---

## 10. Folder structure

```text
D:\druk-league\
├── assets/
│   └── drukleague-logo.png
├── deploy/
│   ├── .env                    # local/private; do not share
│   ├── .env.example
│   ├── docker-compose.yml
│   └── nginx.conf
├── docs/
│   ├── architecture-decision-mvp.md
│   ├── change-control.md
│   ├── defect-log.md
│   ├── risk-register.md
│   ├── stakeholder-analysis.md
│   ├── swe302-test-plan.md
│   ├── traceability-matrix.md
│   └── wbs.md
├── proto/
│   └── drukleague/v1/
│       ├── events.proto
│       ├── match.proto
│       ├── score.proto
│       ├── team.proto
│       ├── tournament.proto
│       └── user.proto
├── services/
│   └── api-gateway/
│       ├── cmd/api/
│       │   ├── admin_matches.go
│       │   ├── events.go
│       │   └── main.go
│       ├── migrations/001_init.sql
│       ├── Dockerfile
│       ├── go.mod
│       └── go.sum
├── web-preview/                # restricted frontend copy for local static preview
│   ├── assets/drukleague-logo.png
│   ├── api.js
│   ├── app.js
│   ├── index.html
│   └── styles.css
├── api.js
├── app.js
├── buf.gen.yaml
├── buf.yaml
├── index.html
├── progress.md
├── README.md
└── styles.css
```

Root also holds `DrukLeague_Project_Proposal.docx`, `DrukLeague_SWE303_Project_ProposalV1.docx`, `Proposal Template.docx` and `DrukLeague_Redesigned_v2.pptx`.

**Target layout after the split (proposed):**

```text
services/
├── api-gateway/        # real gateway (auth, routing, rate limits)
├── bff/
├── <service-name>/     # one folder per service, hexagonal: cmd/, internal/{domain,app,ports,adapters}/, migrations/, Dockerfile, tests
│   └── ...             # auth, user, team, player, tournament, registration, match, score,
│                       # fixture, standings, venue, official, dzongkhag, notification,
│                       # media, report, search, analytics, audit-log, config
functions/              # OpenFaaS/faasd functions (e.g. notify-on-match-finalized)
packages/               # shared proto-generated code, logging, middleware
.github/workflows/      # CI: lint, test, coverage gate, scans
```