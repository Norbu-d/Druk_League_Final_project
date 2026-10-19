# DrukLeague — Project Progress and Handoff

**Last updated:** 2026-10-19
**Project root:** `D:\druk-league`
**Run state:** Containers were last verified on 2026-10-14. On 2026-10-19, Parts 3, 4 and 7 (and the shared part of 6) were written and **tested locally with `go test`**, but **no Docker image has been rebuilt since**. The Dockerfile and Compose changes in section 10 must be made before building. Docker Desktop's disk image is now on **D:** (moved by Norbu on 2026-10-19).

> **Read this first.** The code is a working **MVP vertical slice** (one Go API + plain HTML/CSS/JS frontend) plus 20 service modules. As of 2026-10-19, **14 of those modules have domain logic and unit tests** (Match, Score, Standings, Fixture, Venue, Official, User, Auth, plus the shared packages), Dzongkhag/Team/Tournament/Player/Registration have untested handlers, and 7 are still health shells. The proposal and deck describe a larger system (20 microservices + gateway + BFF + serverless functions). Section 2 lists the differences; section 10 lists what changed on 2026-10-19. Do not present the MVP as the finished architecture.

---

## 1. Project overview

DrukLeague is a tournament operations platform for Bhutan's inter-dzongkhag sports competitions. It replaces phone calls, paper score sheets and messaging apps with one place to create tournaments, register teams, generate fixtures, submit live scores, finalise results and publish standings.

**Group of four:** Tandin Wangchuk, Norbu Dendup, Sonam Choki, Namgay Lhamo.

| Module | Tutor | What it assesses here |
|---|---|---|
| WEB303 Microservices & Serverless | Mr. Pema Namgay | 20 services, gRPC, NATS, OpenFaaS functions |
| SWE302 Software Testing & QA | Mr. Pema Namgay | 80% unit-test coverage per service, CI gates, security/performance tests |
| SWE303 Software Project Management | Mr. Nima Yoezer | Scrum (8 two-week sprints), risks, test plan, traceability |

**Roles (target):** tournament admin (Super Admin in the code), match official, team user, viewer (no login). There is no Dzongkhag Admin role: dzongkhag is reference data (a Dzongkhag service holding the 20 dzongkhags) and a field on teams and tournaments. Authorisation is enforced at the gateway **and** inside each service.

**Core rule:** live scores are mutable; a result becomes final only when the *assigned* match official finalises it. Final results are immutable and are the only source for standings updates and `MatchFinalized` events.

**Budget:** Nu. 0, free/open-source tools, local Docker Compose. Out of scope: production cloud, paid hosting, real SMS/payment, native mobile, production scaling.

**Proposal:** `DrukLeague_Project_Proposal_v2.docx` is current (20 services + gateway + BFF + OpenFaaS, Go, web only) and the deck matches it. `DrukLeague_SWE303_Project_ProposalV1.docx` is an older, smaller draft.

---

## 2. Status vs the proposal

| Proposal / deck says | Code today |
|---|---|
| 20 independently deployable services | 20 modules run as containers (2026-10-14). Since then Fixture, Venue, Official, User and Auth have real hexagonal logic (code + tests, not yet rebuilt in Docker); 7 remain health shells |
| API Gateway + BFF | The monolith is still the "gateway"; no BFF yet (Part 5). Service port 50051 is not published outside Compose |
| gRPC between services | All outbound calls now use `servicekit.Dial`: per-call deadline, retry on safe reads, circuit breaker, request-id and token propagation. Fixture calls Tournament/Team/Venue/Match; Official calls Match; Auth calls User; Venue calls Dzongkhag |
| Each service owns its data | New services have their own databases; the monolith keeps its own shared one; the two are not connected |
| NATS JetStream event flows | Shared `packages/eventkit`: durable consumers with bounded redelivery (5) and a **dead-letter stream `DRUKLEAGUE_DLQ`**, tested against an embedded JetStream server. Match and Standings consumers use it. Fixture publishes `FixturesGenerated` from its own outbox. Notification/Audit consumers not written yet |
| OpenFaaS functions | None |
| OpenTelemetry, circuit breaker, fault isolation | Circuit breaker, deadlines, retries, request-id logging and panic recovery are in `servicekit` (unit-tested). OpenTelemetry not started. Team → Tournament reference check still **open** |
| 80% unit-test coverage, CI gates | Unit + PostgreSQL integration tests for 8 services and the shared packages (section 10). 12 modules still have no tests; no CI workflow yet |
| Hexagonal architecture (cited in the literature review) | Used in Dzongkhag, Match, Score, Standings, Fixture, Venue, Official, User, Auth. Team, Tournament, Player, Registration and the monolith still need extraction |
| JWT + **bcrypt** (deck) | **Services now use bcrypt (cost 12) and `golang-jwt/v5`.** Old PBKDF2 hashes still verify and are upgraded to bcrypt at next login. The monolith still uses its own code until Part 5 replaces it |
| Team users register and track teams | `team_manager` role added in User (Auth/User contracts + DB constraint). Team ownership (`manager_id` on Team) is in the contract but Team does not use it yet |

### The 20 services
Verified = checked in a running container on 12 Oct 2026 with an authenticated gRPC call. All modules have their own contract and Dockerfile.

| Owner | Service | Status |
|---|---|---|
| Tandin | Auth | **Implemented 19 Oct (tested, not containerised):** Login, RefreshToken (rotating), RevokeSession, login lockout (5 failures / 15 min) |
| Tandin | User | **Rewritten 19 Oct (tested, not containerised):** hexagonal, bcrypt, `VerifyCredentials`, bootstrap super admin from `BOOTSTRAP_ADMIN_*`, `team_manager` role, self/admin/service read rules |
| Tandin | Team | Create/Get/List; `CreateTeam`, `GetTeam`, `ListTeams` verified. Does not validate `tournament_id` |
| Tandin | Player | Create/Get/List/Update; `ListPlayers` verified (empty); others not exercised |
| Tandin | Tournament | Create/Get/List; full Create → Get → List round trip verified |
| Tandin | Registration | Register/Review/List; `ListRegistrations` verified (empty); others not exercised |
| Norbu | Match | Verified 14 Oct. 19 Oct: consumer moved to `eventkit` (dead-letter), resilient clients, UUID validation in the app layer, unit + DB tests |
| Norbu | Score | Verified 14 Oct. 19 Oct: event now carries `finalized_by` and `reason`; scores capped 0–99; outbox relay unit-tested |
| Norbu | Standings | Verified 14 Oct. 19 Oct: **fixed bug** where the service token expired 10 min after start (all Team lookups then failed); consumer on `eventkit` |
| Norbu | Fixture | **Implemented 19 Oct (tested, not containerised):** round-robin / double round-robin, venue rotation, creates matches in Match, resumable, `FixturesGenerated` outbox |
| Norbu | Venue | **Rewritten 19 Oct (tested, not containerised):** hexagonal, admin-only create, dzongkhag checked via Dzongkhag service (cached), unique name per dzongkhag |
| Sonam | Dzongkhag | Get/List; `ListDzongkhags` verified (all 20 districts) |
| Sonam | Official | **Rewritten 19 Oct (tested, not containerised):** assigns through Match (source of truth), records tournament, filter by tournament now works |
| Sonam | Notification, Audit Log, Media | Shells |
| Namgay | Report, Search, Analytics, Configuration | Shells |

Part 3 (Fixture, Venue, Official) is written and unit-tested; it still needs the Docker build and container checks in section 10.

---

## 3. Stack

- **Frontend:** plain HTML/CSS/JS, no build step.
- **Backend:** Go. Monolith in `services/api-gateway`; 20 service modules under `services/`.
- **Data:** PostgreSQL 17 via `pgx/v5`; monolith has one startup migration; new services have their own databases.
- **Auth:** services sign/verify HS256 JWTs with `golang-jwt/v5` (same claims as the monolith, so monolith tokens still work) and hash passwords with bcrypt. The monolith still uses its hand-rolled JWT/PBKDF2 code.
- **Contracts:** Protobuf + gRPC for 20 services plus `MatchFinalized`; Buf 1.73.0 lint/build/generate pass.
- **Events:** Score owns the result/outbox transaction and publishes to JetStream. Match and Standings consume the event; duplicate delivery was verified. The monolith's own outbox remains separate until gateway/API integration is changed.
- **Deployment:** Docker Compose (PostgreSQL, NATS, Nginx, API, `db-init` job, 20 services); `deploy/build-services.ps1` builds images.

---

## 4. File guide

- **Frontend:** root `index.html`, `styles.css`, `app.js`, `api.js` are the source of truth; `web-preview/` is a copy, do not edit it.
- **Monolith:** `services/api-gateway/cmd/api/main.go` (HTTP, auth, tournaments, teams, matches, fixtures, scoring, finalisation), `admin_matches.go` (accounts, official assignment, match creation), `events.go` (outbox → JetStream), `migrations/001_init.sql` (schema + trigger blocking edits to finalised results).
- **Contracts and shared code:** `proto/` (layout: `proto/drukleague/v1/` holds events, match, score, team, tournament, user; every other service has `proto/drukleague/<name>/v1/<name>.proto`), `packages/contracts/` (generated Go), `packages/servicekit/` (migration, bearer-token check, gRPC runtime), `services/_template/` (hexagonal example), `services/<service>/` (one per service).
- **Deploy:** `deploy/docker-compose.yml`, `build-services.ps1`, `init-databases.sh`, `nginx.conf`, `.env.example` (variables: `POSTGRES_PASSWORD`, `JWT_SECRET`, `BOOTSTRAP_ADMIN_USER`, `BOOTSTRAP_ADMIN_PASSWORD`). `deploy/.env` is private: never print, copy or commit it. `.gitignore` and `.gitattributes` exist (secrets ignored; shell scripts, Go, SQL, proto, YAML, Dockerfile and nginx config forced to LF).
- **Docs (`docs/`):** `architecture-decision-mvp.md`, `service-design.md` (hexagonal layout and resilience plan), `stakeholder-analysis.md`, `wbs.md`, `risk-register.md`, `change-control.md`, `swe302-test-plan.md` (planned, not executed), `defect-log.md`, `traceability-matrix.md` (user stories and acceptance criteria).
- **Presentation:** `DrukLeague_Redesigned_v2.pptx` (19 slides, speaker notes; Tandin 1–4, Norbu 5–9, Sonam 10–13, Namgay 14–19).

### Monolith behaviour
- Bearer-token auth; the Super Admin creates match officials and viewers; team and tournament writes are admin-only.
- Match creation rejects same-kickoff team/venue clashes; round-robin generation uses a per-tournament advisory lock and does not duplicate.
- Only the assigned official can update or finalise a score. In the **monolith**, finalisation writes the result, both standings rows and the outbox event in one transaction; a trigger rejects later edits.
- On startup the API creates a super-admin only if none exists, so changing the bootstrap password in `.env` later does not change an existing admin.

### Split-service finalisation consistency

The monolith's single transaction cannot span the Score and Standings databases. In the split architecture, Score commits the final result and its outbox event atomically. Standings updates by consuming `MatchFinalized`, so standings are eventually consistent with the final result. The consumer deduplicates by `event_id`, which Score includes in the event payload. Match retains `scheduled`/`completed` status and marks a match completed when it consumes `MatchFinalized`. These flows and duplicate handling were verified on 14 Oct 2026.

| Route | Purpose |
|---|---|
| `POST /api/auth/login` | log in |
| `/api/users` | list/create accounts (Super Admin) |
| `/api/tournaments`, `/api/teams` | create/list |
| `/api/matches` | create/list; `/{id}/official`, `/{id}/live`, `/{id}/finalize` |
| `/api/tournaments/{id}/fixtures` | generate round-robin fixtures |
| `/api/standings?tournament_id=...` | read standings |
| `/healthz` | health |

**Security notes:** replace the custom JWT/PBKDF2 code with a vetted library and bcrypt/argon2; enforce role and ownership checks in every service; clients must never set their own role.

---

## 5. Runtime verification and Docker rules

### Verified 12 Oct 2026 (manual gRPC checks, not automated tests)
Method: a throwaway `fullstorydev/grpcurl` container on the Compose network `deploy_default`, with `proto/` mounted at `/proto` and a bearer token from the login endpoint.

| Check | Result |
|---|---|
| `ListDzongkhags` | All 20 dzongkhags returned |
| `ListTournaments`, `ListTeams`, `ListUsers`, `ListPlayers`, `ListRegistrations` | `{}` (running, authenticated, reading their own database) |
| Call without a token | `Unauthenticated: bearer token required` |
| Tournament Create → Get → List | Persisted; same id and fields; status defaults to `registration_open`; `dzongkhag` is free text |
| Team Create → Get | Persisted with server-side `createdAt` |
| Team Create with a nonexistent `tournamentId` | **Succeeded** (test row "Ghost FC"): Team still accepts this reference; Match now rejects scheduling that team in another tournament |

Not yet exercised: `UpdatePlayer`, `RegisterTeam`, `ReviewRegistration`, `CreatePlayer`, `GetPlayer`, `GetDzongkhag`, Score outbox recovery after a NATS outage, and dead-letter behavior. Test rows ("Test Cup 2026", "Thimphu FC", "Ghost FC", plus the Score verification team, official and match) remain in service databases; there is no Delete RPC, so removing them needs SQL.

### Verified 14 Oct 2026 (manual gRPC and JetStream checks; no automated tests)

- Match, Score and Standings Go builds and Docker builds succeeded; all three containers restarted and ran.
- `CreateMatch` rejected Ghost FC when paired with a team from another tournament (`InvalidArgument`). Match calls Team.GetTeam and User.GetUser with the incoming bearer token; Match creation and official assignment require `super_admin`, and assignment verifies the user's `match_official` role.
- Added persistent verification records because these services have no Delete RPC: a second team in the existing test tournament, one match official in User, and one match. Match assignment succeeded.
- The assigned official changed the live score to 2–1, finalized it, and read the finalized 2–1 through `GetScore`. An admin score update returned `PermissionDenied`; a post-finalization score update returned `AlreadyExists`.
- This was the first live NATS JetStream run. Score's event included an `event_id`; Match consumed it and changed status to `completed`; Standings consumed it and looked up both teams' names and dzongkhag values through Team.GetTeam.
- Republished the same event payload with a different NATS message ID but the same `event_id`. `GetStandings` still showed each team with `played: 1`, confirming duplicate delivery did not double-count.
- Score `GetScore`, Standings `GetStandings`, Match `GetMatch`, and authenticated `ListMatches` returned successfully. NATS outage/reconnect and dead-letter behavior were not tested.

### Docker rules
- **Docker Desktop's disk image was moved to D: on 2026-10-19** (Settings > Resources > Advanced). On 19 Oct, D: had about 122 GB free and C: about 23 GB. Keep at least 10 GB free on both drives. Earlier, a full C: drive (Docker's `docker_data.vhdx` was 25.8 GB) caused the hangs and HTTP 500s.
- Start with `docker compose up -d --no-build`. Build one service at a time: `$env:COMPOSE_PARALLEL_LIMIT = 1`, then `docker compose build <service>` (repeat the `$env:` line in each new PowerShell window).
- After a restart the engine may return HTTP 500 for a minute or two. Wait 60–90 seconds, or run `wsl --shutdown` and reopen Docker Desktop.
- Pruning frees space inside Docker's disk file; the file itself only shrinks after compacting (`diskpart`, `compact vdisk`, with Docker and WSL stopped).
- **Never run** `docker volume prune`, `docker system prune --volumes` or `docker compose down -v`. Keep the volumes `drukleague-postgres-data` (PostgreSQL 16, preserved), `drukleague-postgres17-data` (used by Compose) and `deploy_nats-data`.
- Minimum machine for teammates: about 10 GB free disk and 8 GB RAM.

---

## 6. Open decisions

1. **Password hashing:** *resolved in code 19 Oct:* the services use bcrypt, matching the deck. The monolith still uses PBKDF2 until Part 5 retires it.
2. **Team-user role:** *started 19 Oct:* role name `team_manager` in code. Align the proposal and deck wording, and finish Team ownership and Registration rules.
3. **OpenFaaS vs faasd:** OpenFaaS normally needs Kubernetes; faasd runs on one machine. Pick one and spike it early.
4. **Risk R1:** the deck rates "20 services in one semester" High/High; the proposal says Medium. Make them match.
5. **Finalised-result correction:** admin correction with a reason + audit entry (user stories) vs append-only correction events (architecture note).
6. **Sprint 4/5 dependency:** fixture clash detection needs Venue, planned for Sprint 5. Reorder or stub Venue earlier.
7. **NATS resilience:** bounded redelivery and dead-lettering are implemented in `eventkit` and pass against an embedded JetStream server. Still test NATS outage/reconnect in Docker, and write the Notification/Audit consumers.
8. **Hexagonal and resilience wording:** the proposal cites hexagonal architecture but never says it will be applied. Add one methodology sentence and a resilience objective, then update the deck's architecture and testing slides, the risk table and the speaker notes.
9. **Cross-service integrity:** Team stores any `tournament_id`. Choose a synchronous gRPC check against Tournament (with deadline, retry, circuit breaker) or eventual consistency with a cleanup event. Record the finding in the risk register either way.

Decided: build toward all 20 services (7 Oct 2026); Dzongkhag Admin role dropped (7 Oct 2026).

---

## 7. Remaining work (priority order)

0. **Next (before anything else):** do the section 10 "before building" steps, then rebuild and container-check User, Auth, Venue, Official, Fixture, Match, Score and Standings one at a time.
1. **Docker disk:** done. The disk image is on D: (19 Oct).
2. **Push to GitHub (before Part 2).** Remaining checks before the first commit: `git check-ignore -v deploy/.env` names a rule; list staged files over 1 MB across the whole repo; grep staged files for secret-looking assignments (file names only); set `git config user.name` and `user.email`. Then commit, create a **private** repository, push, add the three teammates and protect `main`.
3. **Secrets before sharing:** rotate everything in `deploy/.env`. `JWT_SECRET` is safe to change on its own (recreate the API afterwards). `POSTGRES_PASSWORD` and `BOOTSTRAP_ADMIN_PASSWORD` need a coordinated change (PostgreSQL reads its password only when the volume is created; the bootstrap admin exists once created), so choose `ALTER USER` plus an admin reset, or a clean dev reset. Fix the README's outdated default-login line. Bind DB/API ports to loopback for local runs.
4. **Part 2:** Match, Score and Standings are extracted. Verified Match → Team/User, Score → Match, Score transactional outbox → JetStream, Match completion consumer, Standings consumer, GetScore/GetStandings and duplicate-event handling. Standings is eventually consistent with Score; no distributed transaction spans their databases.
5. **Part 3:** code and tests done (19 Oct); container checks pending. **Part 4:** User + Auth code and tests done (19 Oct); container checks pending. **Part 5 (next to write):** a BFF (REST → gRPC, same routes as the monolith so the frontend keeps working) and the gateway; point nginx at it. Also fix finalize in the BFF so it uses the score the official submits (the monolith ignores the body and finalizes the last saved live score).
6. **Part 6:** `eventkit` (redelivery limit, dead-letter stream) and the Fixture outbox are done. Still to do: Notification and Audit Log consumers, and the OpenFaaS/faasd function.
7. **Part 7:** deadlines, retries on safe reads, circuit breaker, panic recovery and request-id logging are done in `servicekit`. Still to do: the Team → Tournament reference check, and running the fault-isolation scenarios in Docker.
8. **Part 8:** the remaining 7 shells: Notification, Audit Log, Media, Report, Search, Analytics, Configuration.
9. **Part 9 (SWE302):** unit tests with port fakes and PostgreSQL integration tests exist for 8 services plus the shared packages (section 10). Still to do: tests for the 12 remaining modules and the monolith; fix `scripts/coverage.sh` on Windows; GitHub Actions with lint, tests and a coverage gate; gosec, govulncheck, Trivy, ZAP; k6; Playwright; fault-isolation tests.
10. **Frontend:** replace sample data with API data (dashboard, all-tournament standings, officials, venues, reports); team-user and public viewer views; loading/error/empty states.
11. **Observability:** OpenTelemetry traces across gateway and services.
12. **SWE303 evidence:** sprint plans, reviews, retrospectives; update the defect log, risk register (add the cross-service finding) and traceability matrix with executed results; RACI/estimation/communication plan; final report and demo script.
13. **Housekeeping:** README quickstart for teammates (clone, copy `.env.example`, set secrets, `docker compose up -d --no-build`, one-at-a-time builds); Buf breaking check needs a baseline revision; optionally publish images to GitHub Container Registry later.

**Sprint mapping (proposal's 8 sprints):** S1 foundation/CI/contracts · S2 identity + gateway + BFF · S3 teams/tournaments/registration · S4 competition (match → standings) · S5 venues/officials/notifications · S6 supporting services · S7 integration, events, serverless, observability, security/performance tests · S8 QA, coverage, defects, demo.

---

## 8. Continue in a new Claude chat

Give the new chat `progress.md`, `README.md` and `docs/service-design.md`. If it cannot see the project folder, also paste the files for the part you are on (for Part 2: `services/api-gateway/cmd/api/main.go`, `admin_matches.go`, `events.go`, `migrations/001_init.sql`, and the `services/dzongkhag` folder as the reference structure). Then send:

> DrukLeague is a Go microservices project targeting all 20 services. Read progress.md, README.md and docs/service-design.md first. Never ask me to paste, and never print, deploy/.env.
>
> **How we work:** I use Windows, PowerShell and VS Code. You cannot run anything on my machine. Give me copy-paste-ready code (full file replacements, not diffs, with brief inline comments) and PowerShell commands. I paste the code into VS Code, run `go build`, the tests and the Docker commands myself, and send you the output. Before writing code that changes an existing file, ask me to paste that file; do not guess what is in it. Give me one next step at a time and wait for my output before moving on. When I paste terminal output, I paste only the output, not old prompts or errors.
>
> **Status:** see progress.md section 10 (19 Oct 2026). Parts 3 and 4 and the shared resilience/event code are written and unit-tested but not yet rebuilt in Docker; do the section 10 "before building" steps first. Follow the Docker rules in progress.md section 5: build one service at a time, keep 10 GB free on C:, and never run `docker volume prune` or `down -v`. Never print or share `deploy/.env`.
>
> **Task:** we are working through section 7 part by part. Start by telling me which step is next and what you need from me. Do not claim tests or features work unless I have shown you the output.

---

## 9. Folder structure

As of 12 Oct 2026, taken from the files Git is tracking. `deploy/.env` exists locally but is private and not committed.

```text
D:\druk-league\
├── .dockerignore
├── .gitattributes
├── .gitignore
├── assets/
│   └── drukleague-logo.png
├── deploy/
│   ├── .env                      # local/private; never share or commit
│   ├── .env.example
│   ├── build-services.ps1
│   ├── docker-compose.yml
│   ├── init-databases.sh
│   └── nginx.conf
├── docs/
│   ├── architecture-decision-mvp.md
│   ├── change-control.md
│   ├── defect-log.md
│   ├── risk-register.md
│   ├── service-design.md
│   ├── stakeholder-analysis.md
│   ├── swe302-test-plan.md
│   ├── traceability-matrix.md
│   └── wbs.md
├── packages/
│   ├── contracts/
│   │   ├── gen/drukleague/       # generated Go/gRPC code, one folder per service
│   │   ├── go.mod
│   │   └── go.sum
│   └── servicekit/
│       ├── cmd/runner/main.go
│       ├── servicekit/
│       │   ├── auth.go           # bearer-token check
│       │   └── run.go            # migration + gRPC runtime
│       ├── go.mod
│       └── go.sum
├── proto/drukleague/
│   ├── v1/                       # events, match, score, team, tournament, user
│   │   └── *.proto
│   └── <name>/v1/<name>.proto    # analytics, audit_log, auth, configuration, dzongkhag,
│                                 # fixture, media, notification, official, player,
│                                 # registration, report, search, standings, venue
├── services/
│   ├── _template/                # hexagonal example every service copies
│   │   ├── cmd/service/main.go
│   │   ├── internal/
│   │   │   ├── adapters/postgres/ (errors.go, records.go)
│   │   │   ├── app/records.go
│   │   │   ├── domain/record.go
│   │   │   └── ports/record_repository.go
│   │   ├── migrations/001_init.sql
│   │   ├── Dockerfile, go.mod, go.sum
│   ├── api-gateway/              # the monolith API
│   │   ├── cmd/api/
│   │   │   ├── admin_matches.go
│   │   │   ├── events.go
│   │   │   └── main.go
│   │   ├── migrations/001_init.sql
│   │   ├── Dockerfile, go.mod, go.sum
│   ├── dzongkhag/                # reference implementation (full hexagonal layout)
│   │   ├── cmd/service/main.go
│   │   ├── internal/
│   │   │   ├── adapters/grpc/server.go
│   │   │   ├── adapters/postgres/repository.go
│   │   │   ├── app/service.go
│   │   │   ├── domain/dzongkhag.go
│   │   │   └── ports/repository.go
│   │   ├── migrations/001_init.sql
│   │   ├── Dockerfile, go.mod, go.sum
│   ├── service-builder/Dockerfile   # shared build image
│   ├── service-runtime/Dockerfile   # shared runtime image
│   └── <service>/                # the other 18 services, each with:
│       ├── cmd/service/main.go   #   cmd/service/main.go, Dockerfile, go.mod, go.sum
│       └── internal/adapters/grpc/server.go   # present for match, official, player,
│                                              # registration, team, tournament, user, venue
├── web-preview/                  # copy of the frontend for the local preview; do not edit
│   ├── api.js
│   ├── app.js
│   ├── index.html
│   └── styles.css
├── api.js                        # frontend source of truth
├── app.js
├── buf.gen.yaml
├── buf.yaml
├── index.html
├── progress.md
├── README.md
└── styles.css
```

The 20 services are: auth, user, team, player, tournament, registration, match, score, fixture, standings, venue, official, dzongkhag, notification, media, report, search, analytics, audit-log, configuration.

Root also holds the proposal and deck files: `DrukLeague_Project_Proposal_v2.docx`, `DrukLeague_Project_Proposal.docx`, `DrukLeague_SWE303_Project_ProposalV1.docx`, `Proposal Template.docx` and `DrukLeague_Redesigned_v2.pptx`.

**Target layout after the split (proposed):**

```text
services/
├── api-gateway/          # real gateway (auth, routing, rate limits)
├── bff/
└── <service>/            # hexagonal: cmd/, internal/{domain,app,ports,adapters}/, migrations/, Dockerfile, tests
functions/                # OpenFaaS/faasd functions (e.g. notify-on-match-finalized)
.github/workflows/        # CI: lint, test, coverage gate, scans
```

---

## 10. Session log — 19 Oct 2026

### What changed

**Shared packages**
- `packages/servicekit`
  - JWTs now use `golang-jwt/v5`. Tokens the monolith already issues still verify.
  - **Bug fixed:** `Login` and `RefreshToken` were never public. The check looked for `/AuthService/Login`, but the real method name is `/drukleague.auth.v1.AuthService/Login`.
  - New `Dial` client: per-call deadline (3 s), retry with backoff on `Get`/`List`/`Search`/`Verify` calls only, a circuit breaker per target (opens after 5 failures, 10 s cool-down), request-id propagation, and token forwarding or a service token.
  - New `TokenSource` that re-signs service tokens before they expire.
  - Server side: panic recovery and one log line per RPC with its request id.
  - Shared `ToStatus`/`FromStatus` error mapping, plus `Page`, `Env` and `Secret` helpers.
  - `ApplyMigrations` is now exported.
- `packages/servicekit/core`: dependency-free role names, UUID helpers and shared domain errors.
- `packages/servicekit/dzongkhags`: cached Dzongkhag lookup that keeps using its stale cache while the Dzongkhag service is down.
- `packages/servicekit/testdb`: gives each integration test a throwaway PostgreSQL database; skipped unless `TEST_DATABASE_URL` is set.
- `packages/eventkit` (new): stream setup (legacy stream subjects upgraded in place), a dead-letter stream `DRUKLEAGUE_DLQ`, durable consumers with a redelivery limit and backoff, and a de-duplicated publisher.

**Contracts** (all additive; regenerated with Buf 1.73.0 in Docker; `buf lint` passes)
- User: new `VerifyCredentials` RPC.
- Auth: `refresh_token` and `session_id` in the login and refresh responses.
- `MatchFinalized`: `finalized_by` and `reason`.
- Team: `manager_id` (plus a filter for it).
- Fixture: `match_id`, `round` and `starts_at`.
- Report: `tournament_id` and `format`.
- Analytics: venue, official and pending-registration counts.

**Services**
- **Match:** consumer moved to `eventkit`; completion goes through the app layer and repository; resilient clients; UUID validation.
- **Score:** event payload carries the actor and reason; scores capped 0–99; outbox SQL moved into the repository and the relay unit-tested.
- **Standings:** **bug fixed:** its service token expired 10 minutes after start. Win/draw/loss rule moved into the domain; a team that does not exist goes to the dead-letter stream instead of retrying.
- **Fixture:** see section 2. Migration `002_schedule.sql`.
- **Venue:** see section 2. Migration `002_unique_name.sql`.
- **Official:** see section 2. Migration `002_tournament.sql`.
- **User:** see section 2. Migration `002_roles.sql`, which adds the role CHECK and case-insensitive unique usernames. **It fails if the existing User database has two usernames differing only by case.**
- **Auth:** see section 2. Migration `002_sessions.sql`.
- Every service module now has the `contracts` replace directive.
- The Match/Score/Standings `vendor/` folders (stale copies of `servicekit`/`contracts`) were moved out of the repo into a scratchpad backup.

### Test coverage (19 Oct 2026, `go test -cover`, with PostgreSQL integration tests enabled)

Coverage is per package (statements). `cmd/` wiring is not tested.

| Module | domain | app | gRPC server | gRPC clients | NATS | PostgreSQL |
|---|---|---|---|---|---|---|
| Match | via app | 100% | 98.2% | 100% | 100% | 84.7% |
| Score | via app | 100% | 100% | 100% | 100% | 84.8% |
| Standings | 100% | 100% | 100% | 100% | 100% | 80.6% |
| Fixture (93.6% overall) | 100% | 93.9% | 100% | 100% | 100% | 78.9% |
| Venue | via app | 100% | 95.2% | — | — | 92.0% |
| Official | — | 100% | 100% | 100% | — | 87.5% |
| User | via app | 96.4% | 96.2% | hasher 100% | — | 87.2% |
| Auth | via app | 90.0% | 100% | 100% | — | 97.1% |

| Shared package | Coverage |
|---|---|
| servicekit | 86.1% |
| servicekit/core | 94.1% |
| servicekit/dzongkhags | 100% |
| eventkit | 89.2% (includes an embedded-JetStream dead-letter test) |

**No tests yet (0%):** Dzongkhag, Team, Tournament, Player, Registration, Notification, Audit Log, Media, Report, Search, Analytics, Configuration, and the monolith. `scripts/coverage.sh` (per-module total) prints FAIL on Windows Git Bash even though the tests pass, most likely its temp-file path. Fix it before using it as a CI gate.

How to run the integration tests locally (throwaway in-memory database, port 55432, no volume):

```powershell
docker run -d --rm --name drukleague-testdb -e POSTGRES_PASSWORD=test -p 127.0.0.1:55432:5432 --tmpfs /var/lib/postgresql/data postgres:17-alpine
$env:TEST_DATABASE_URL = "postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable"
cd services\match; go test -cover ./...
```

### Before building (not done yet — Docker builds of these services will fail until it is)
1. `.dockerignore`: allow `packages/eventkit/` and `packages/eventkit/**`, the same way `packages/servicekit` is allowed.
2. Dockerfiles for match, score, standings and fixture: add `COPY packages/eventkit/ /src/packages/eventkit/`.
3. Rebuild the shared builder image, because `servicekit` now depends on `contracts` and `golang-jwt`: `docker build --tag drukleague-service-builder:local --file services/service-builder/Dockerfile .`
4. `deploy/docker-compose.yml`:
   - **user:** `BOOTSTRAP_ADMIN_USER` and `BOOTSTRAP_ADMIN_PASSWORD`.
   - **auth:** `USER_SERVICE_ADDR`, plus optionally `ACCESS_TOKEN_TTL` and `REFRESH_TOKEN_TTL`.
   - **venue:** `DZONGKHAG_SERVICE_ADDR`.
   - **official:** `MATCH_SERVICE_ADDR`.
   - **fixture:** `TOURNAMENT_/TEAM_/VENUE_/MATCH_SERVICE_ADDR` and `NATS_URL`, plus `depends_on: nats`.
5. Then build and check one service at a time (section 5 rules).

### Verified in containers — 19 Oct 2026 (after the session above)
- The "before building" steps 1–4 were done: `.dockerignore` and every Dockerfile include `packages/eventkit`, the builder image was rebuilt, and Compose has the new environment variables. All 20 service images were rebuilt one at a time; all 25 containers started. The User migration `002_roles.sql` applied cleanly, and the User service logged `bootstrap super admin created`.
- New end-to-end smoke test `tools/e2e` (Go, runs inside the Compose network). Result: **29 of 29 checks passed**:
  - Login, wrong password, refresh rotation and reuse rejection, lockout after 5 failures, and logout.
  - Role rules for User, Venue and Score.
  - Venue dzongkhag checked through the Dzongkhag service.
  - Fixture generation (6 fixtures for 4 teams, each linked to a real match, re-run creates nothing), with the tournament dzongkhag's venue.
  - Official assignment via Match, and filtering by tournament.
  - Scoring limits, finalisation and immutability.
  - Match and Standings consuming `MatchFinalized` (the winner gets 3 points, team names come from Team).
  - A completed match cannot be reassigned.
- Run it:

```powershell
docker run --rm --network deploy_default -v D:/druk-league:/src -w /src/tools/e2e -e E2E_ADMIN_PASSWORD=<your BOOTSTRAP_ADMIN_PASSWORD> drukleague-service-builder:local go run .
```

- Each run leaves uniquely named test rows (a tournament, 4 teams, a venue, an official and 6 matches), because there are no delete RPCs.
