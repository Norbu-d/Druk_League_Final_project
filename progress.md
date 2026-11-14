# DrukLeague — Project Progress and Handoff

**Last updated:** 2026-11-12
**Project root:** `D:\druk-league`
**Run state:** The 19 Oct full Compose run passed 29/29 E2E checks (section 10). On 11 Nov, the recovered Docker WSL disk started the local Compose stack; 26 long-running containers were up, PostgreSQL and NATS were healthy, and the web page returned HTTP 200 (section 40). This was a startup smoke check, not a rerun of the 29/29 E2E suite. PR #31 merged into `development` on 11 Nov with 17/17 checks passing; local `development` was fast-forwarded to that merge and is clean. PR #2 remains open and unmergeable: its feature branch has a merge conflict with `development`, and its latest SonarCloud check reports Security Rating B (required A). Commit `5d92461` on `feature/norbu-match-engine` contains the BFF cookie-auth and Sonar findings follow-up; its focused Go, JavaScript, PowerShell, Compose-config, and diff checks passed locally. The fixes are not yet integrated into `development`. Sonam's PR #11 for the media schema passed all checks and merged to `development` on 5 Nov 2026; SCRUM-83 is Done. OpenTelemetry tracing remains deferred per SCRUM-72. The active Docker disk is now under `D:\docker\DockerDesktopRecovery\DockerDesktopWSL`.

> **Read this first.** The code is a working **MVP vertical slice** (one Go API + plain HTML/CSS/JS frontend) plus 20 service modules. The Report, Search query, Audit Log, Analytics event aggregation/dashboard query, Configuration RPCs, and Media upload/retrieval have implementation beyond health shells; several need more acceptance checks or tests. Several services and the monolith still need more tests. The proposal and deck describe a larger system (20 microservices + gateway + BFF + OpenFaaS functions). Section 2 lists the differences; section 10 lists the 2026-10-19 container run; section 11 holds the 2026-10-22 design review.

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
| 20 independently deployable services | 20 modules run as containers (2026-10-14). Fixture, Venue, Official, User, Auth, Team, Notification and Report have real service logic; five modules remain health shells |
| API Gateway + BFF | The monolith is still the "gateway"; no BFF yet (Part 5). Service port 50051 is not published outside Compose |
| gRPC between services | All outbound calls now use `servicekit.Dial`: per-call deadline, retry on safe reads, circuit breaker, request-id and token propagation. Fixture calls Tournament/Team/Venue/Match; Official calls Match; Auth calls User; Venue calls Dzongkhag |
| Each service owns its data | New services have their own databases; the monolith keeps its own shared one; the two are not connected |
| NATS JetStream event flows | Shared `packages/eventkit`: durable consumers with bounded redelivery (5) and a **dead-letter stream `DRUKLEAGUE_DLQ`**, tested against an embedded JetStream server. Match, Standings, and Notification consume `MatchFinalized`; Fixture publishes `FixturesGenerated` from its own outbox. Audit Log consumer remains |
| OpenFaaS functions | None |
| OpenTelemetry, circuit breaker, fault isolation | Circuit breaker, deadlines, retries, request-id logging and panic recovery are in `servicekit` (unit-tested). OpenTelemetry not started. Team → Tournament reference check still **open** |
| 80% unit-test coverage, CI gates | GitHub Actions enforces the 80% coverage threshold on modules with tests. Unit + PostgreSQL integration tests cover 11 services and shared packages; four CRUD modules, five shell services, and the monolith still need tests |
| Hexagonal architecture (cited in the literature review) | Used in Dzongkhag, Match, Score, Standings, Fixture, Venue, Official, User, Auth. Team, Tournament, Player, Registration and the monolith still need extraction |
| JWT + **bcrypt** (deck) | **Services now use bcrypt (cost 12) and `golang-jwt/v5`.** Old PBKDF2 hashes still verify and are upgraded to bcrypt at next login. The monolith still uses its own code until Part 5 replaces it |
| Team users register and track teams | `team_manager` role and Team `manager_id` ownership are implemented; managers can create/read only their own teams |

### The 20 services
Verified = checked in a running container on 12 Oct 2026 with an authenticated gRPC call. All modules have their own contract and Dockerfile.

| Owner | Service | Status |
|---|---|---|
| Tandin | Auth | **Implemented 19 Oct (tested; containerised and e2e-checked 19 Oct):** Login, RefreshToken (rotating), RevokeSession, login lockout (5 failures / 15 min) |
| Tandin | User | **Rewritten 19 Oct (tested; containerised and e2e-checked 19 Oct):** hexagonal, bcrypt, `VerifyCredentials`, bootstrap super admin from `BOOTSTRAP_ADMIN_*`, `team_manager` role, self/admin/service read rules |
| Tandin | Team | Create/Get/List; `CreateTeam`, `GetTeam`, `ListTeams` verified. Does not validate `tournament_id` |
| Tandin | Player | Create/Get/List/Update; `ListPlayers` verified (empty); others not exercised |
| Tandin | Tournament | Create/Get/List; full Create → Get → List round trip verified |
| Tandin | Registration | Register/Review/List; `ListRegistrations` verified (empty); others not exercised |
| Norbu | Match | Verified 14 Oct. 19 Oct: consumer moved to `eventkit` (dead-letter), resilient clients, UUID validation in the app layer, unit + DB tests |
| Norbu | Score | Verified 14 Oct. 19 Oct: event now carries `finalized_by` and `reason`; scores capped 0–99; outbox relay unit-tested |
| Norbu | Standings | Verified 14 Oct. 19 Oct: **fixed bug** where the service token expired 10 min after start (all Team lookups then failed); consumer on `eventkit` |
| Norbu | Fixture | **Implemented 19 Oct (tested; containerised and e2e-checked 19 Oct):** round-robin / double round-robin, venue rotation, creates matches in Match, resumable, `FixturesGenerated` outbox |
| Norbu | Venue | **Rewritten 19 Oct (tested; containerised and e2e-checked 19 Oct):** hexagonal, admin-only create, dzongkhag checked via Dzongkhag service (cached), unique name per dzongkhag |
| Sonam | Dzongkhag | Get/List; `ListDzongkhags` verified (all 20 districts) |
| Sonam | Official | **Rewritten 19 Oct (tested; containerised and e2e-checked 19 Oct):** assigns through Match (source of truth), records tournament, filter by tournament now works |
| Namgay | Notification | Persistent NATS consumer and in-app result notices implemented in SCRUM-18 |
| Sonam | Media | SCRUM-83 schema designed and merged in PR #11; upload/retrieval remains under SCRUM-84 |
| Namgay | Audit Log | Implementation completed and merged under SCRUM-68 (section 19) |
| Namgay | Report | Persistent standings and fixture CSV/JSON exports implemented in SCRUM-65; Sonam's SCRUM-85 schema subtask is Done |
| Sonam | Search, Configuration | Search query endpoint and configuration schema merged; indexing/read-write follow-ups remain under SCRUM-87 and SCRUM-94 |
| Sonam | Analytics | SCRUM-89 schema and SCRUM-90 MatchFinalized consumer are Done; query/service endpoints remain open |
| Sonam/Norbu | Analytics | Sonam's SCRUM-90 consumer persists finalized-match facts; Norbu added filtered dashboard counts (SCRUM-67). Broader service-owned totals and current runtime RPC acceptance remain open. |

Part 3 (Fixture, Venue, Official) is written, unit-tested, containerised and covered by the 19 Oct e2e run.

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

- **Frontend:** root `index.html`, `styles.css`, `app.js`, `api.js` are the source of truth. The unused `web-preview/` duplicate was removed on 6 Nov 2026.
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
3. **OpenFaaS vs faasd:** OpenFaaS normally needs Kubernetes; faasd runs on one machine. Pick one and spike it early. **22 Oct recommendation:** faasd alongside Compose, no Kubernetes; confirm with Mr. Pema Namgay in writing (section 11.7).
4. **Risk R1:** the deck rates "20 services in one semester" High/High; the proposal says Medium. Make them match.
5. **Finalised-result correction:** admin correction with a reason + audit entry (user stories) vs append-only correction events (architecture note). **22 Oct recommendation:** admin-only void-and-replace with an audit record (section 11.4).
6. **Sprint 4/5 dependency:** fixture clash detection needs Venue, planned for Sprint 5. Reorder or stub Venue earlier.
7. **NATS resilience:** bounded redelivery and dead-lettering are implemented in `eventkit` and pass against an embedded JetStream server. Still test NATS outage/reconnect in Docker and implement the Audit Log consumer.
8. **Hexagonal and resilience wording:** the proposal cites hexagonal architecture but never says it will be applied. Add one methodology sentence and a resilience objective, then update the deck's architecture and testing slides, the risk table and the speaker notes.
9. **Cross-service integrity:** Team stores any `tournament_id`. Choose a synchronous gRPC check against Tournament (with deadline, retry, circuit breaker) or eventual consistency with a cleanup event. Record the finding in the risk register either way. **22 Oct:** the cleaner fix is to move the tournament link from Team to Registration (item 10, section 11.3).
10. **Registration model (proposed 22 Oct):** Registration links Team to Tournament with a status; Team becomes a persistent entity; Fixture reads approved registrations. Needs agreement with Tandin (section 11.3).
11. **Service identity and user → official mapping (proposed 22 Oct):** a distinct service claim instead of a borrowed `super_admin`, and an explicit mapping from user ID to official ID. Needs agreement with Tandin and Sonam (section 11.2).
12. **Scope cuts (proposed 22 Oct):** drop or document OpenTelemetry, five shell services and asymmetric JWT signing. Reverses part of the 7 Oct "all 20 services" decision, so it needs a group decision (section 11.1).

Decided: build toward all 20 services (7 Oct 2026); Dzongkhag Admin role dropped (7 Oct 2026).

---

## 7. Remaining work (priority order)

> The 22 Oct design review proposes a revised order in section 11.1. Where it differs from the list below, section 11.1 wins until the group agrees otherwise.

0. **Done (19 Oct):** the section 10 "before building" steps, the one-at-a-time rebuild of all 20 images, and the 29/29 e2e run. Next: section 11.1.
1. **Docker disk:** done. The disk image is on D: (19 Oct).
2. **GitHub collaboration:** the repository exists and changes have been pushed. Confirm teammate access and protect `main`; the screenshot on 9 Nov showed `main` unprotected. `deploy/.env` is ignored by `.gitignore`.
3. **Secrets before sharing:** rotate everything in `deploy/.env`. `JWT_SECRET` is safe to change on its own (recreate the API afterwards). `POSTGRES_PASSWORD` and `BOOTSTRAP_ADMIN_PASSWORD` need a coordinated change (PostgreSQL reads its password only when the volume is created; the bootstrap admin exists once created), so choose `ALTER USER` plus an admin reset, or a clean dev reset. The README documents the configured bootstrap login; local web/API ports are being bound to loopback.
4. **Part 2:** Match, Score and Standings are extracted. Verified Match → Team/User, Score → Match, Score transactional outbox → JetStream, Match completion consumer, Standings consumer, GetScore/GetStandings and duplicate-event handling. Standings is eventually consistent with Score; no distributed transaction spans their databases.
5. **Part 3:** code, tests and container checks done (19 Oct). **Part 4:** User + Auth, same. **Part 5:** a BFF and nginx cutover, including finalising the score submitted by the official, are implemented on `feature/norbu-match-engine` (PR #2). They are not yet integrated into `development`; PR #2 currently conflicts with `development` and its latest SonarCloud Security Rating is B (required A). Resolve both before counting Part 5 as complete.
6. **Part 6:** `eventkit` (redelivery limit, dead-letter stream), Fixture outbox, Notification consumer and Audit Log consumers are implemented. Still to do: OpenFaaS/faasd function (pending tutor confirmation) and live Docker outage/reconnect/replay acceptance.
7. **Part 7:** deadlines, retries on safe reads, circuit breaker, panic recovery and request-id logging are done in `servicekit`. Team → Tournament reference validation is implemented on `feature/norbu-match-engine`/PR #2 but not yet integrated into `development`; Docker fault-isolation scenarios remain.
8. **Part 8:** Media upload/retrieval implementation exists (SCRUM-64/84); authenticated browser E2E and restore-from-backup remain. Search synchronization design (SCRUM-87) is documented, while indexing rollout remains. Configuration RPCs (SCRUM-94) and Analytics dashboard metrics (SCRUM-67/90) are implemented, with live acceptance still pending. Audit Log admin-action publishing and its end-to-end flow passed CI; SCRUM-68 is Done.
9. **Part 9 (SWE302):** GitHub Actions runs Go formatting, vet, build, tests, module coverage, Buf, govulncheck, gosec, Trivy and k6. PR #31 passed 17/17 checks before merging (11 Nov). Remaining: reconcile the 80% coverage gate and its module results; add test suites for modules without tests and the monolith; ZAP; browser E2E/Playwright; Docker fault-isolation and recovery acceptance. PR #2's SonarCloud check currently fails the required Security Rating A gate (latest rating B). SCRUM-55 remains open until its full criteria and current evidence are reconciled.
10. **Frontend:** replace sample data with API data (dashboard, all-tournament standings, officials, venues, reports); team-user and public viewer views; loading/error/empty states.
11. **Observability:** OpenTelemetry traces across gateway and services.
12. **SWE303 evidence:** sprint plans, reviews, retrospectives; update the defect log, risk register (add the cross-service finding) and traceability matrix with executed results; RACI/estimation/communication plan; final report and demo script.
13. **Housekeeping:** README quickstart covers clone/configure/start and one-at-a-time builds. A local PostgreSQL plus media backup script and restore guide are in place; live backup/restore acceptance remains. The Buf breaking check uses the target branch history. Confirm teammate access and branch protection; optionally publish images to GitHub Container Registry later.

**Sprint mapping (proposal's 8 sprints):** S1 foundation/CI/contracts · S2 identity + gateway + BFF · S3 teams/tournaments/registration · S4 competition (match → standings) · S5 venues/officials/notifications · S6 supporting services · S7 integration, events, serverless, observability, security/performance tests · S8 QA, coverage, defects, demo.

---

## 8. Continue in a new Claude chat

Give the new chat `progress.md`, `README.md` and `docs/service-design.md`. If it cannot see the project folder, also paste the files for the part you are on (for Part 2: `services/api-gateway/cmd/api/main.go`, `admin_matches.go`, `events.go`, `migrations/001_init.sql`, and the `services/dzongkhag` folder as the reference structure). Then send:

> DrukLeague is a Go microservices project targeting all 20 services. Read progress.md, README.md and docs/service-design.md first. Never ask me to paste, and never print, deploy/.env.
>
> **How we work:** I use Windows, PowerShell and VS Code. You cannot run anything on my machine. Give me copy-paste-ready code (full file replacements, not diffs, with brief inline comments) and PowerShell commands. I paste the code into VS Code, run `go build`, the tests and the Docker commands myself, and send you the output. Before writing code that changes an existing file, ask me to paste that file; do not guess what is in it. Give me one next step at a time and wait for my output before moving on. When I paste terminal output, I paste only the output, not old prompts or errors.
>
> **Status:** see progress.md sections 10 and 11 (19 and 22 Oct 2026). Parts 3 and 4 and the shared resilience/event code are written, unit-tested, containerised and e2e-checked (29/29); section 11 holds planned design changes, none implemented yet. Follow the Docker rules in progress.md section 5: build one service at a time, keep 10 GB free on C:, and never run `docker volume prune` or `down -v`. Never print or share `deploy/.env`.
>
> **Task:** we are working through section 11.1 (then section 7) step by step. Start by telling me which step is next and what you need from me. Do not claim tests or features work unless I have shown you the output.

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
│   ├── nginx.conf
│   └── nginx-security.conf
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

**Tests still needed:** Dzongkhag, Tournament, Player, Registration, Audit Log, Media, Search, Analytics, Configuration, and the monolith. Team, Notification, and Report now have service tests; CI provides PostgreSQL for their integration tests and enforces the coverage gate.

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

---

## 11. Design review — 22 Oct 2026

**Status: partially addressed.** The original findings below were a design review, not an implementation checklist. Sections 12–38 record follow-up work and evidence. Several items are now implemented, while those marked *group decision* still need owner/tutor agreement. Do not treat a recommendation as an approved group decision. Where this section conflicts with the ordering in section 7, this section wins until the group agrees otherwise.

### 11.1 Proposed order of work

1. **Tag the known-good state.** Commit and create a git tag on the 19 Oct state (29/29 e2e checks) so there is something to return to.
2. **Write three short design notes (about half a page each) and agree them with the group:** the registration model (11.3), the result correction path (11.4), and service identity plus the user → official mapping (11.2). Tandin owns Team, Registration and Auth; Sonam owns Official.
3. **Registration and match fixes** (11.3, 11.4): move the tournament link to Registration with a status; Fixture reads approved registrations; finalise carries the score or a version and rejects a mismatch; add an official-overlap check in Match. Start with Match, Score and Fixture (Norbu's services) and hand Tandin the Registration contract change.
4. **Minimum gateway and frontend cutover.** Not the full BFF: a small Go gateway that validates the JWT and exposes REST for one flow end to end (login, list tournaments, scoring and finalisation). Switch the frontend over flow by flow and keep the monolith as a fallback. This is what makes the microservices visible in a demo.
5. **CI and coverage gate** (11.5, 11.6): fix `scripts/coverage.sh` first, then one GitHub Actions workflow running `go test -cover` per module and failing under 80%.
6. **Tests for Team, Tournament, Player and Registration**, after step 3 so they cover the new registration design.
7. **If time remains:** Notification and Audit Log consuming `MatchFinalized`; one or two OpenFaaS/faasd functions (11.7).

**Proposed to cut, or to document as a known limitation** (*group decision*, because 7 Oct 2026 decided to build all 20 services): OpenTelemetry, the then-shell services, and asymmetric JWT signing. Report has since been implemented; the active shell list is in section 2.

### 11.2 Auth

- **Shared HS256 secret.** Any service that can verify a token can also mint one, including an admin token. Preferred fix: asymmetric signing (Auth holds the private key, other services get the public key). If that is too large for the timeline, state it as a known limitation in the report.
- **Service identity.** Standings' service token expired after 10 minutes (fixed 19 Oct). Define what claim a service token carries and who mints it. Use a distinct service claim, not a borrowed `super_admin`.
- **User account → Official record.** The `match_official` role lives in User, assignment lives in Official and Match, and finalisation checks that the assigned official is the caller. An explicit mapping from user ID to official ID is needed; none is documented. *Group decision (Sonam).*
- **Role changes are not immediate** if the role is carried in the JWT: a demoted user keeps access until the token expires. Keep the access TTL short and rely on refresh.
- **Lockout can be abused.** Five failures per 15 minutes per username lets anyone lock the admin out. Key it on username plus IP, or use backoff instead of a hard lock.
- **Two issuers.** Services accept monolith tokens. Fine during migration; retire it at cutover (monolith still uses hand-rolled JWT/PBKDF2 until Part 5).
- **Account lifecycle.** No change-password, admin reset or deactivate RPC. Changing the bootstrap password in `.env` does not change an existing admin.
- **Ownership.** `manager_id` is in the Team contract but unused, so any team manager can act on any team.

### 11.3 Registration model

- **The link is on the wrong side.** Team carries `tournament_id` and a separate Registration service exists. Proposed: Registration links a Team to a Tournament with a status (`pending`, `approved`, `rejected`, `withdrawn`); Team becomes a persistent entity (dzongkhag, sport, manager) reused across seasons instead of duplicated per tournament. This also resolves open decision 9 (Team stores any `tournament_id`).
- **Fixture should read approved registrations**, after registration closes, instead of teams filtered by tournament. That gives a clear cutoff and a rule for late teams (11.4).
- **Rules to define:** registration deadline, capacity, one team per dzongkhag per sport, roster lock once the tournament starts.
- **Events:** `RegistrationApproved` and `RegistrationClosed`, so Fixture and Notification can react instead of polling.
- Needs a contract and migration change in Team and Registration. *Group decision (Tandin).*

### 11.4 Fixtures and matches

- **No correction path for a wrong result.** Finalised results are immutable (a trigger blocks edits) and Standings only reacts to `MatchFinalized`. Proposed: an admin-only void-and-replace flow that emits `MatchResultCorrected` with an audit record; Standings reverses the old result and applies the new one. This is open decision 5; the trigger and the Score immutability rule need an explicit, audited exception.
- **Finalise ignores the request body** and finalises the last saved live score, so an in-flight update can produce the wrong final score. Have finalise carry the score or a version number and reject a mismatch (also noted for the BFF in section 7, step 5).
- **Too few match states.** Only `scheduled` and `completed` exist. Postponed, cancelled, forfeit and abandoned will occur and need standings rules.
- **Fixture edge cases.** Odd team counts need byes. Define what happens when a team withdraws or joins after generation (re-running creates nothing, so it does not answer this).
- **Official clashes.** Match checks team and venue clashes but no check for the same official on overlapping matches is documented. Confirm that matches have a duration, because the clash checks depend on it.
- **Scoring and standings scope.** The 0–99 score and win/draw/loss points suit football, not set-based sports or knockout formats. State the sport scope and add tie-breakers (goal difference, head-to-head).
- **Time zone.** Bhutan is UTC+6. Store `starts_at` in UTC and convert at the gateway or frontend.

### 11.5 Operations and hardening

- **`buf breaking` in CI** so a renamed or removed field fails the build (needs a baseline revision, section 7 step 13).
- **Security scans in CI:** `govulncheck`, `gosec`, Trivy for images.
- **Compose health.** Separate liveness from readiness (can it reach its DB and NATS?) and use `depends_on: condition: service_healthy` for PostgreSQL, NATS and `db-init`, to stop startup races.
- **Backups.** A `pg_dump` script covering every service database, run before demos, instead of relying on "never run `down -v`".
- **Standings rebuild.** A replay command that recomputes standings from finalised results in Score. Also makes the correction flow simpler.
- **Dead-letter stream inspection/replay.** Implemented in `tools/event-admin`; entries remain retained for 30 days after replay.
- **Retention.** Clean up published outbox rows and set retention limits on the JetStream streams.
- **Double-submit protection.** Idempotency keys or natural-key unique constraints on create calls (register team, finalise).
- **Test data.** The e2e run leaves rows behind. Prefix test data and add a cleanup path, or run e2e against a throwaway database.
- **Migration risk.** User `002_roles.sql` fails if two usernames differ only by case. Add a pre-check or dedupe step.

### 11.6 Frontend, edge and report evidence

- **nginx:** rate limits on login and public viewer reads, CORS, basic security headers.
- **Token storage.** Check where the plain-JS frontend keeps the JWT. Prefer the access token in memory and the refresh token in an httpOnly cookie if the gateway can set it; otherwise document it as a limitation.
- **Session lifetime.** The browser session uses an HttpOnly, SameSite=Strict cookie with a 12-hour lifetime. A rotating refresh-token flow remains open.
- **List endpoints.** Make sure every List RPC (tournaments, teams, matches, standings) uses `Page` and supports filters.
- **SWE302:** a performance test (k6 or `hey`) against the gateway for login and the public standings read; traceability from each requirement to a test and a sprint.
- **Audit Log** is the first shell worth building for real: admin actions, role changes and result corrections.

### 11.7 Deployment: Kubernetes vs OpenFaaS

These are not alternatives. Kubernetes would orchestrate the 20 services; OpenFaaS runs functions and can run without Kubernetes as faasd on one machine.

- **Recommendation:** keep Docker Compose for the 20 services and do not adopt Kubernetes. The machine is already tight on RAM and disk, production cloud and scaling are out of scope, and a cluster would cost a large share of the remaining time without a visible gain in the demo.
- Add faasd and deploy one or two small functions (for example one triggered by `MatchFinalized` that formats a notification, or one that generates a standings report), so the serverless claim is true.
- In the report, list Kubernetes as future work. Each service already has its own Dockerfile and env-var config, so the move would mostly be manifests.
- **Before committing:** ask Mr. Pema Namgay in writing whether faasd on Docker is acceptable for the serverless component. If the module spec requires Kubernetes, this recommendation changes. Tracks open decision 3.

---

## 12. SCRUM-44 — Fixture consumes finalized-match events (implemented 2 Nov 2026)

- Fixture now consumes `MatchFinalized` using its own durable JetStream consumer, independently of the other consumers. It records a match as completed through the Fixture application/repository path.
- Unit coverage was added for the consumer and service flow. The end-to-end smoke program was extended to check completion after finalisation; the existing 29/29 container run predates this extension, and the extended full Compose flow has not been rerun locally because Docker Desktop is unavailable.
- Work is in the `feature/namgay-insights-testing` history and was included in PR #3, merged to `development` on 2 Nov 2026.

## 13. SCRUM-18 — Notification service (end-to-end verification follow-up, 6 Nov 2026)

Jira assigns SCRUM-18 to Namgay Lhamo in Sprint 6, *Notify & Media* (due 14 Dec 2026), and its parent status is Done. SCRUM-42 (event listener) and SCRUM-43 (notification dispatch) are also Done. PR #25 merged to `development` on 6 Nov 2026 (merge commit `98b6e0f`); the notification smoke test passed in CI. The stated requirement is a persistent NATS subscriber that independently consumes `MatchFinalized` and sends result notifications to affected teams/dzongkhags; it must be a microservice, not a serverless function.

- Added a persistent Notification gRPC service backed by PostgreSQL, with `SendNotification` and paginated `ListNotifications` RPCs. A user can list their own notices; administrators can list notices for other users.
- Added a separate durable JetStream consumer for `drukleague.match.finalized`. It validates events, retries transient failures through `eventkit`, routes invalid events to the permanent-failure path, and deduplicates notices by event, recipient and channel.
- On finalisation, the service looks up both teams through authenticated Team gRPC calls and creates one in-app notice for each distinct team manager. The result message includes both team names, scores, dzongkhag names and finalisation time. The existing model has no dzongkhag account identity or recipient mapping, so notices are delivered to team managers rather than directly to district accounts.
- Added domain, application, gRPC, consumer and PostgreSQL integration tests, plus end-to-end smoke assertions. PR #3 merged the implementation to `development` on 2 Nov 2026. `go test ./...` in `services/notification` passes locally on 6 Nov.
- Strengthened `tools/e2e` to check that participating team managers receive the correct final score, team names, and affected dzongkhag names, while unrelated team managers receive no notice. Added a CI job to start the Compose dependencies (including Dzongkhag) and run this smoke flow; the CI E2E passed on 6 Nov. Corrected a stale standings export assertion to check the actual finalized match participants. Local Docker verification remains unavailable because Docker Desktop's Linux engine is not running. PR #25 is merged. The current user model has no dzongkhag account identities, so delivery is to participating team managers and names both affected dzongkhags in the notice.

## 14. SCRUM-65 — Report service (implemented; Jira Done)

- Added a persistent Report gRPC service that reads standings and fixtures through authenticated service-to-service gRPC and stores report metadata and export bytes in PostgreSQL.
- Supports CSV and JSON exports for standings and fixtures, authenticated create/get/download HTTP routes, owner/admin access, pagination, and CSV formula-injection mitigation. The API gateway's Nginx configuration routes report HTTP requests to the service.
- Added service unit, gRPC, HTTP and PostgreSQL integration tests, and end-to-end smoke assertions for report creation and download. The local full Compose run remains unavailable; the CI suite provides database-backed coverage.
- Committed as `b27d9a6` on `feature/namgay-insights-testing` and merged to `development` by PR #10 at merge commit `241e735`. All checks passed for the report code commit; Jira SCRUM-65 is Done. The report database schema subtask SCRUM-85 was confirmed against the merged migrations and marked Done on 5 Nov 2026.

## Sonam — Earlier merged contributions

- **SCRUM-18 / PR #1:** contributed the initial Notification service and MatchFinalized result-notice implementation on `feature/sonam-platform-services`; the Jira story's later ownership and follow-up are described in section 13.
- **SCRUM-45 / PR #4:** implemented standings-seeded knockout brackets, byes, and automatic next-round fixture generation after finalized results. Scoped fixture-pair uniqueness by format and addressed CI security findings. PR checks passed and the PR merged.
- **SCRUM-54 / PR #5:** added and expanded Dzongkhag, Player, and Tournament tests, including validation, CRUD, pagination, and error cases; addressed Buf setup and coverage findings. All PR checks passed and it merged.
- **SCRUM-48 / PR #6:** added role-based frontend route guards with redirect behavior and session role restoration; verified Viewer restrictions and Super Admin access, addressed Sonar findings, and merged after checks passed.
- **SCRUM-88 / SCRUM-66 / PR #7:** implemented the Search gRPC query endpoint with validation, team/player/tournament filters, and pagination. Replaced dynamic SQL with fixed parameterized queries after the security scan. PR checks passed and it merged; search indexing/data synchronization remains tracked separately by SCRUM-87.
- **SCRUM-93:** added and verified the Configuration typed-metadata schema migration; the existing key/value contract remains compatible. The schema change merged; read/write endpoints remain tracked by SCRUM-94.

## 15. Sonam — Prometheus and Grafana monitoring (SCRUM-72, 2026-11-04)
- Added gRPC request counters and latency histograms to the shared service runtime, Prometheus scrape targets for all 20 services, and a provisioned Grafana dashboard.
- Rebuilt and started the Compose stack; Prometheus showed 20/20 targets UP and the dashboard displayed metrics after gRPC E2E traffic.
- The REST API gateway accesses PostgreSQL directly, so normal UI actions do not create gRPC metric samples. OpenTelemetry tracing remains deferred per the Jira scope.
- Added tests for Prometheus metrics formatting and gRPC request recording; the full `bash scripts/coverage.sh 80` check passed locally with PostgreSQL integration tests enabled, and `packages/servicekit` reached 85.2% coverage.
- PR #9 passed all checks and merged to `development`; Jira SCRUM-72 is Done. OpenTelemetry tracing remains deferred per the issue scope.

## 16. Sonam — Media-service database schema (SCRUM-83, 2026-11-04)
- Added an additive migration for match-linked media assets with photo/highlight type, caption, optional duration, and an index for listing a match's media by creation time.
- Kept `match_id` as an opaque UUID because Match owns a separate database; a check constraint requires it for new rows while allowing existing rows to be migrated.
- Applied the initial and metadata migrations to a temporary PostgreSQL database; reapplied the metadata migration successfully to verify idempotency, then dropped the test database.
- Committed as `2189db9`; merged development with conflict resolution in `0ad6f41`. PR #11 passed all checks and merged to `development` on 5 Nov 2026. Jira SCRUM-83 is Done; no sprint dates changed.

## 17. Sonam — Analytics-service database schema (SCRUM-89, 2026-11-05)

- Added `services/analytics/migrations/002_match_facts.sql` with a finalized-match facts table, unique event and match IDs for deduplication, bounded score checks, season/dzongkhag dimensions, and indexes for tournament, season/dzongkhag, and team queries. The existing `analytics_snapshots` table remains unchanged.
- Applied migrations `001` and `002` to a temporary PostgreSQL database; reapplied `002` successfully to verify idempotency, then dropped the temporary database.
- Committed as `115f7c1` on `feature/sonam-platform-services`; the PR passed all checks and merged to `development`. Jira SCRUM-89 is Done. Analytics event aggregation and service implementation remain tracked separately under SCRUM-90.

## 18. Progress update practice

Whenever a Jira story is completed, add a dated entry here with its key, implemented behavior, test/verification evidence, commit or PR state, and Jira status. Keep unverified local runs and known requirement gaps explicit; do not mark the Jira story Done while a required child item or acceptance requirement remains open.

## 19. SCRUM-68 — Audit log service (completed 6 Nov 2026)

- Jira shows SCRUM-68 in Sprint 7 (*Search & Config*), assigned to Namgay Lhamo and due 28 Dec 2026. SCRUM-68 and its subtasks SCRUM-91 (database schema) and SCRUM-92 (audit event recording) are Done.
- Implemented persistent audit-entry storage and idempotency for event-backed records, an authenticated gRPC record/list API, admin-only audit reads, and durable consumers for `MatchFinalized` and the `drukleague.admin.*` event family.
- Added the audit-entry schema migration and wired the audit-log service to PostgreSQL and JetStream. The existing `RecordAuditEntry` RPC derives the actor from the authenticated identity rather than trusting the request actor field.
- 6 Nov follow-up: extended the API gateway producer path to audit successful admin account, team, tournament, match, official-assignment, and fixture-generation actions, alongside Norbu's result-correction event. Each event includes event ID, authenticated actor ID, action, resource type/ID, reason or context, and UTC occurrence time; it is inserted into the existing outbox in the same transaction as the corresponding change. Added unit coverage for accepted outbox subjects and rejected wildcard/untrusted subjects. `gofmt`, `git diff --check`, and local `go test ./internal/auditoutbox` pass. All GitHub CI checks passed, including Go tests/coverage, security scans, protobuf, performance, SonarCloud, container image build/scan, and the API-to-NATS-to-Audit-Log-Postgres end-to-end test. The E2E check signed in as the bootstrapped admin, created a tournament, and confirmed its actor/action/resource audit row in PostgreSQL. PR #23 was merged into `development` at `5bffe6626d25d8eb51d95e60679d93e6c064528e`; SCRUM-91, SCRUM-92, and SCRUM-68 were then transitioned to Done in Jira.

## 20. SCRUM-57 — Project charter (draft completed 5 Nov 2026)

- Added `docs/project-charter.md` covering purpose, measurable objectives, in/out of scope, deliverables, stakeholders, governance, milestones, assumptions, constraints, risks, and acceptance/review roles for DrukLeague.
- Kept milestone dates out of the charter because the Jira sprint dates are being corrected; Jira remains the schedule source. The document distinguishes the 20-service target from the staged implementation and does not claim unreceived approvals.
- Reviewed against the existing WBS, stakeholder analysis, architecture decision, risk register, and project overview; `git diff --cached --check` passed before commit. The charter was committed as `4afa7b6`, documented in `83f955a`, and merged by PR #14 into `development` at merge commit `0dac320` on 5 Nov 2026. All 11 PR checks passed. SCRUM-57 and SCRUM-25 were transitioned to Done in Jira on 5 Nov 2026 after confirming both SCRUM-25 subtasks are Done and the architecture documentation exists. SCRUM-68 was completed on 6 Nov 2026 (see section 19).

## 21. Sonam — Analytics event aggregation (SCRUM-90, merged 5 Nov 2026)

- Added a durable `MatchFinalized` consumer that validates event IDs, match/team IDs, score bounds, and finalization time; it looks up tournament season and dzongkhag using the Analytics service identity and writes each finalized match to `analytics_match_facts` idempotently.
- Added Analytics Compose settings for NATS and Tournament service addresses. `go mod tidy`, `go build ./...`, and the Analytics Docker image build succeeded. Analytics, NATS, and Tournament ran successfully; the analytics database contained 4 match facts with 4 unique event IDs and match IDs after the consumer processed existing finalized-match events. Committed as `3bcfda2` and merged to `development`; Jira SCRUM-90 is Done.

## 18. SCRUM-49 — Role-based component visibility (merged, 5 Nov 2026)

- Jira assigns SCRUM-49 to Namgay Lhamo in Sprint 8, *Integration & QA*. The requirement names only Super Admin, Match Official, Team Manager and Viewer; Dzongkhag Admin was dropped.
- Extended the frontend's existing role-aware navigation to hide controls by role: tournament and fixture management for Super Admin, score updates for Match Officials, team registration for Team Managers, and read-only behavior for Viewers. Administrative official/venue controls are hidden from non-admin roles. Export and read-only match details remain available to the four roles.
- Added role checks to UI click and form submission paths so manually invoked hidden actions are rejected with a user-facing message. The authenticated API/server remains the security boundary; these browser checks are presentation behavior only.
- Updated `app.js` and `api.js`. No local tests or build checks were run. Committed as `55dc688` and `22210cc`, then merged to `development` in PR [#16](https://github.com/Norbu-d/Druk_League_Final_project/pull/16) at merge commit `666d197` on 5 Nov 2026. All 11 CI checks passed. Jira was initially To Do and Namgay confirmed it was changed to Done on 5 Nov 2026.

## 21. SCRUM-24 — Testing pyramid (Namgay contribution merged, 6 Nov 2026)

- Jira shows the parent assigned to Namgay in Sprint 8 (*Integration & QA*), In Progress, with two of three children Done. SCRUM-55, integration and contract tests, remains In Progress and assigned to Norbu.
- Extracted the four-role page/action permission policy into `role-permissions.js`, added four Node unit tests for page access, fallback roles, mutation permissions and read-only export, and added a CI job to run them. Updated `docs/swe302-test-plan.md` with case T-17 and its evidence.
- Added Audit Log unit tests for event validation and conversion, authorization and pagination, JetStream message handling, and gRPC request/response mapping. Added PostgreSQL repository tests for event idempotency, filtered/paginated reads, and database errors; these run automatically in CI's PostgreSQL-backed test job.
- PR [#26](https://github.com/Norbu-d/Druk_League_Final_project/pull/26) merged to `development` at `0362590` on 6 Nov. All PR checks passed, including Go lint/build/test/80% coverage, security, image scan, notification and audit E2E, frontend permissions, protobuf, and performance.
- Local verification on 6 Nov: the audit-log Go packages pass, and combined `internal/...` coverage is 80.6% against the 80% gate. PostgreSQL integration cases were skipped locally because `TEST_DATABASE_URL` is unset. The previously added four Node permission tests and syntax checks passed. The full Compose E2E remains assigned to SCRUM-56 (Done); cross-service integration and contract tests remain in Norbu's open SCRUM-55.
- SCRUM-24 remains open until the full story scope and SCRUM-55 are complete and CI/E2E acceptance evidence is reconciled. SCRUM-68 is Done.

## 22. SCRUM-103 — Nginx hardening (in progress, 6 Nov 2026)

- Added login and public-standings rate limits, baseline security headers, and a local-origin CORS allowlist to the Nginx edge; removed the unused `web-preview/` duplicate. The API gateway no longer emits a wildcard CORS origin.
- `docker compose config --quiet` passed. `nginx -t` passed in `nginx:1.27-alpine` using temporary `api` and `report` hostname aliases. Docker Desktop was unavailable for an end-to-end Compose run at the first implementation pass.
- Moved the API JWT out of `sessionStorage` into an HttpOnly, SameSite=Strict cookie; added session restore and logout endpoints and changed the frontend to use same-origin requests. The API gateway build, `node --check` for the frontend scripts, `docker compose config --quiet`, and containerized `nginx -t` passed. End-to-end browser/session verification was not run; no rotating refresh-token flow exists. Keep SCRUM-103 In Progress pending acceptance review.

## 23. SCRUM-102 — Dead-letter inspection and replay (implementation, 6 Nov 2026)

- Added `tools/event-admin inspect` to list retained DLQ metadata and `replay <sequence>` to republish a message to its recorded original subject. `--dry-run` validates the target; replay retains the source DLQ record.
- The tool build succeeded. The shared DLQ stream already has a 30-day retention policy. Live replay against a running NATS service remains to be verified before closing the Jira issue.
- Added daily cleanup of published transactional outbox rows older than 30 days in API Gateway, Score, and Fixture; pending rows are never deleted. Live verification remains pending.
- The JetStream event stream now has 365-day retention (DLQ remains 30 days), and startup reconciles existing streams to those limits as well as adding missing subjects.

## 24. SCRUM-87 — Search index and synchronization design (6 Nov 2026)

- Documented the source-of-truth boundary, entity document shape, versioned outbox events, idempotent consumers, snapshot-plus-stream rebuild procedure, eventual consistency, query limits, and operational metrics in `docs/search-index-design.md`.
- Confirmed the Search gRPC query service and migration already exist. Entity event producers, snapshot APIs, and index population are rollout dependencies and are not claimed complete by this design task.

## 25. SCRUM-94 — Configuration read/write gRPC endpoints (implementation, 9 Nov 2026)

- Added authenticated `GetConfiguration` and super-admin-only `SetConfiguration` RPC handlers backed by the configuration service database. Keys are validated and values are bounded to 64 KiB; current contract stores values as strings, matching the existing proto key/value shape.
- Wired the service into the shared gRPC server. Build and runtime verification are pending.

## 26. SCRUM-101 — Standings rebuild from finalized events (implementation, 9 Nov 2026)

- Added `services/standings/cmd/rebuild`, which replays retained `MatchFinalized` events, reloads Team metadata, deduplicates by event ID, and replaces standings plus processed-event IDs atomically.
- The command holds an exclusive lock on `processed_events` while capturing the JetStream tail and rebuilding, so the live consumer waits and resumes afterward. It refuses an empty or truncated event history.
- `go mod tidy` and `go build ./...` succeeded for the Standings module. The rebuild ran against the Compose stack and replayed 13 event IDs into 26 team rows; a direct database read confirmed 13 processed event IDs, 26 standings rows and 26 total appearances. Jira SCRUM-101 is Done. Rebuild input currently covers `MatchFinalized`; correction-event reconciliation remains a separate known gap.

## 27. Norbu — Analytics dashboard and SWE303 evidence updates (9 Nov 2026)

- Added `AnalyticsService.GetDashboardMetrics` backed by filtered finalized-match facts. The endpoint returns tournament count, distinct participating teams and completed matches. Player, venue, official and pending-registration counts are not fabricated because those source facts are not currently present in the Analytics projection. `go build ./...` and `git diff --check` passed in `services/analytics`; runtime gRPC acceptance remains pending. Commit `4c399d0`; Jira SCRUM-67 remains open.
- Refreshed the defect log, traceability matrix and risk register to remove stale status claims and record current known gaps, cross-service correction risk, and observed Compose/Nginx/event evidence. Commit `2e58a1b`. The sprint plan/review/retrospective, R1 rating and Sprint 4/5 reorder required by SCRUM-108 remain incomplete; keep that issue In Progress.
- These commits are on `codex/norbu-tandin-open-work`, which has been pushed to `origin`. Do not infer that the full assigned backlog is complete from these entries.

## 28. Norbu — Media upload and retrieval API (9 Nov 2026)

- Implemented same-origin authenticated multipart upload, per-match listing, retrieval and uploader/Super Admin deletion in the Media service. The server validates Match UUID format, photo/highlight type and content sniffing, caps files at 10 MiB, and writes random-keyed blobs outside PostgreSQL.
- Added `media-data` persistent Compose volume, Nginx routes and `docs/media-service.md`. `go build ./...`, `docker compose config --quiet`, and the single-service Docker image build passed. Media started in Compose and its `/healthz` returned `{"status":"ok"}`.
- Authenticated upload/download/delete end-to-end, browser workflow and restore-from-backup have not been checked. Jira SCRUM-64 and SCRUM-84 remain In Progress. Commit `97acc50`.

## 29. Norbu — Analytics and Media test coverage additions (9 Nov 2026)

- Added Media PostgreSQL/HTTP integration cases for authenticated upload, per-match listing, retrieval bytes, owner deletion, and invalid/unauthenticated uploads; added Analytics dashboard filter unit and PostgreSQL projection cases. Added T-18/T-19 to `docs/swe302-test-plan.md`.
- The database-backed cases were run against a temporary PostgreSQL 17 container. See section 31 for full-suite results and remaining coverage gaps.

## 30. SCRUM-99 — Fixture uses approved tournament registrations (9 Nov 2026)

- Changed Fixture participant discovery from legacy `Team.tournament_id` membership to paginated Registration service queries filtered to `approved`. Fixture scheduling and knockout generation now use only approved teams; the end-to-end setup registers and approves its generated test teams before fixture creation.
- Added Registration service startup dependencies and address wiring in Compose, and client pagination/outage coverage. `go build ./...` passed for Fixture and `tools/e2e`; Compose configuration validation and `git diff --check` passed. The Fixture container rebuilt and is running.
- Fixture gRPC, app, adapter and client tests passed in the full Go module test run recorded in section 31. Registration withdrawal, registration deadlines/capacity, roster locking and the underlying Team/tournament model migration still require agreed product decisions; SCRUM-99 remains In Progress.

## 31. SCRUM-55 — Integration and contract test verification (9 Nov 2026)

- Fixed a Fixture gRPC test fake left on the old team-list port after Fixture switched to approved Registration entries. Fixed an EventKit test assertion so it checks the Events stream update without assuming the DLQ stream is not also reconciled.
- Ran `go test -work -count=1 ./...` for every Go module using Go 1.26.5 and an isolated PostgreSQL 17 test database; every module passed. Existing service gRPC client/server tests, database-backed tests, NATS tests and the new Media/Analytics tests passed.
- Buf 1.73.0 `lint`, `build`, and `breaking --against .git#branch=development` passed after fetching the current `origin/development` baseline. The run used Go 1.26.5; GitHub Actions pins 1.26.10.
- The CI coverage command was attempted using Go 1.26.10. Its 80% gate remains red for Analytics (10.9%) and Media (63.7%); the API Gateway coverage attempt was blocked by `goproxy.io` returning HTTP 400 for `klauspost/compress`. Audit Log, Configuration, Search and API Gateway currently have no `_test.go` files. SCRUM-55 remains In Progress until these coverage gaps and missing test suites are addressed.
- Focused test fixes are included in the SCRUM-55 branch update.

## 32. SCRUM-21 — Role-based UI (implementation complete, 6 Nov 2026)

- SCRUM-21 is assigned to Namgay. Existing role-based page and action visibility is retained; the sidebar and header show signed-in username and role, and signing out restores Public viewer / Viewer. `role-permissions.js` centralizes role labels, profile formatting, and Viewer fallback.
- The local `file:///` launch uses the API at `127.0.0.1:8080`; failed sign-ins show an inline error. The recommended full-app launch remains Docker Compose at `http://localhost:4173`.
- Verification recorded on the feature branch: all 6 Node role-permission/profile tests passed; `node --check` passed for `role-permissions.js`, `app.js`, and `api.js`; `git diff --check` passed. Browser reproduction remains pending, and backend authorization remains the security boundary.
- PR #28 merged into `development` at `db8e8db`. The last recorded Jira status was To Do; refresh its acceptance criteria and status. SCRUM-24 remains open.

## 33. Sync with `development` (9 Nov 2026)

- Merged the fetched `origin/development` into `codex/norbu-tandin-open-work`. Resolved conflicts in `api.js`, `progress.md`, and API Gateway `events.go`, retaining the HttpOnly cookie session, role-aware profile behavior, stored-subject validation, outbox retention, and both branches' progress records.
- Updated the admin audit end-to-end script to use the HttpOnly session cookie returned by login; the API no longer returns a bearer token. `bash -n`, `node --check api.js`, six role-permission/profile Node tests, Compose config validation, and `git diff --check` passed.
- `go test -work -count=1 ./...` passed across all Go modules; `services/api-gateway` tests also passed. Docker Desktop's Linux engine was unavailable for the isolated PostgreSQL test database, so DB-backed tests were skipped in this combined-tree run. Earlier DB-backed results are recorded in section 31.
- This sync is a local merge commit on the Codex branch; it is not yet merged into `development`. PR/review and CI checks for the combined branch remain required before integrating into the shared branch.

## 34. CI failures on combined branch (9 Nov 2026)

- Reviewed the failed checks for merge commit `3f7b76c`. The Media module-manifest check found `github.com/jackc/pgx/v5` was imported directly but declared indirect; `services/media/go.mod` now declares it directly.
- SonarCloud later flagged that the session and logout cookies did not explicitly set `Secure`. PR #2 follow-up now sets `Secure: true` on both cookies and their clearing headers; local-browser acceptance should confirm the cookies are retained on localhost.
- The Trivy setup action selected `latest` and exited during binary installation without a diagnostic. A pinned binary version also failed silently, so the image scan is being switched to the pinned `aquasec/trivy:0.68.2` container.
- Buf and k6 setup did not reach lint/tests: GitHub release downloads returned HTTP 504 after retries. These failures need a fresh CI run to determine whether they were transient.
- Local `go test ./...` passed for Media and API Gateway, and `git diff --check` passed. Docker-backed tests and the full CI run remain pending.

## 35. Follow-up to CI run on `7cdf494` (9 Nov 2026)

- CI confirmed that the cookie findings are resolved. The security scan then found three Media issues: multipart size parsing, path use from the media metadata key, and file permissions. Media storage now uses `os.Root` operations to confine file access, validates generated hex storage keys, and writes files as owner-only. The request body remains capped at 10 MiB; the narrow G120 suppression documents that bound.
- The pinned Trivy binary download still exited without a diagnostic, so `.github/workflows/go-ci.yml` now runs a pinned Trivy container to scan the built local Docker images.
- The same CI run still reports Analytics coverage at 10.9% and Media at 63.7%, below the existing 80% gate. The Buf and k6 release downloads again returned HTTP 504. Media tests passed after the storage changes; gosec and the updated image scan need a new CI run.

## 36. Coverage gate follow-up (9 Nov 2026)

- Added Analytics tests for finalized-event normalization/validation, service dependency and persistence behavior, gRPC request/response and errors, tournament gRPC client mapping, NATS handling/retry classification, and database idempotency.
- Added Media tests for authentication and cookie fallback, malformed and invalid uploads, content-type validation, invalid resource IDs, storage setup failures, and storage-key helpers.
- Analytics package tests pass locally and its non-database coverage profile is 89.1%. Media tests pass locally; database-backed Media tests are skipped because Docker Desktop's engine is unavailable, so only 40.2% local coverage was measured. Check the CI module coverage result on the pushed branch before claiming the 80% gate is resolved.
- The first CI run with the new Analytics tests reported Analytics at 100%; Go vet caught and then the follow-up commit fixed a protobuf lock copy in the new gRPC client test. The next run showed Media at 71.1%, so additional DB-backed cases now cover missing records/files, invalid storage keys, admin deletion, empty results, and upload cleanup after metadata failure.
- 10 Nov 2026 — SCRUM-67: expanded Analytics dashboard metrics across tournament, team, player, match, venue, official, and registration data. The branch recorded 90.5% internal coverage and an authenticated runtime RPC check.

## 37. Follow-up CI and handoff corrections (9 Nov 2026)

- Commit `fdd5ce7` (`codex/fix-sonar-security-rating`) validates API response identities before browser storage, restricts browser API paths to known same-origin routes, documents the local-development-only cookie Secure exception for SonarCloud, and uses the pinned `bufbuild/buf:1.73.0` container instead of downloading the Buf release binary.
- All 8 GitHub Actions checks on `fdd5ce7` completed successfully, including protobuf, frontend, Go coverage, security, image scan, audit event, and k6 checks. The commit is pushed; no PR has been created or merged yet.
- The Buf container change was verified by its successful GitHub check. Local Docker Desktop's Linux engine is unavailable in this session, so Docker Compose, browser, and fault-recovery acceptance could not be run here.
- Updated the README, defect log, traceability matrix, and test-plan status to reflect implemented Media/Audit Log behavior and the current CI results. Configured the published local API and web ports to bind to loopback in Compose.
- Added `scripts/backup.ps1` and `docs/backup-restore.md` to back up all PostgreSQL databases and uploaded media together; live backup/restore acceptance is still required.
- Still requiring team/member action: coordinate secret rotation against the persistent database and bootstrap account; confirm faasd scope with the tutor; agree Registration, service identity/official mapping, and scope decisions with the named owners; run the live Compose/browser/fault tests; reconcile Jira and branch protection; and record actual sprint review/retrospective input. These decisions and test results must not be invented in the progress record.

## 38. Docker recovery and local operations follow-up (9 Nov 2026)

- Docker Desktop was initially available, but a `docker compose up -d --build` attempt began building all service images concurrently and the Linux engine stopped responding with HTTP 500. The build was interrupted; no volume prune/remove/down command was run.
- Restart attempts left Docker Desktop's service stopped. Only the `docker-desktop` WSL distribution had been running; it was shut down. Starting `com.docker.service` from this session failed with a Windows service access error, so the engine is not currently usable. A local administrator may need to restart Docker Desktop/the service before Compose acceptance can continue.
- Updated `deploy/build-services.ps1` to build each service image in an individual Compose invocation, preventing the full multi-image Bake build from launching at once.
- Added `scripts/backup.ps1` and `docs/backup-restore.md` for PostgreSQL and media-volume backups, with archives stored outside the repository by default. PowerShell syntax parsing and `git diff --check` passed; the live backup/restore procedure has not been executed because Docker is unavailable.
- No database or media volume was deleted or reset. Re-run the sequential image build, `docker compose up -d --no-build`, health checks, authenticated Media workflow, and E2E/fault scenarios after Docker is restarted.

## 39. Docker data disk reattachment required (9 Nov 2026)

- Docker Desktop now responds, but its active engine lists zero volumes and uses a newly initialized disk at `C:\\Users\\DELL\\AppData\\Local\\Docker\\wsl\\disk\\docker_data.vhdx` (about 1.6 GB).
- The original Docker data disk is still present and unchanged at `D:\\docker\\DockerDesktopWSL\\disk\\docker_data.vhdx` (about 41 GB). No Compose stack was started on the new empty disk, to avoid creating a separate empty database state.
- In Docker Desktop, open **Settings → Resources → Advanced → Disk image location** and select the existing disk on D:. If prompted to use the existing image or replace it, choose the existing image. Verify that the prior images and volumes reappear before running Compose. Docker's settings guide documents the disk-location control and the choice when an image already exists.
- This is a historical snapshot from 9 Nov and was superseded by the recovery and Compose startup recorded in section 40.

## 40. Docker recovery, local Compose startup, and branch sync (11 Nov 2026)

- Reattached the Docker data by copying the original 41 GB `docker_data.vhdx` into the configured Recovery location. The original source copy was later removed after the recovered PostgreSQL volume was confirmed usable. The active disk is `D:\docker\DockerDesktopRecovery\DockerDesktopWSL\disk\docker_data.vhdx`.
- The service image build completed after the network changed. `docker compose up -d` then started 26 long-running containers; PostgreSQL and NATS reported healthy, and `http://127.0.0.1:4173` returned HTTP 200. A read-only PostgreSQL check found 8 public tables, 1 user, 13 teams, 2 tournaments, and 78 matches. This is startup/data verification, not full browser or E2E acceptance.
- Notification logs reported MatchFinalized events dead-lettered because both participating teams need assigned managers. Investigate the affected records and verify delivery with valid manager assignments.
- Removed the specifically requested unused Buslink, Food Ordering, and Jenkins volumes after confirming no containers referenced them. Cleared the unused build cache and removed unreferenced Go builder/base images; future image builds may need to download or rebuild these layers again.
- Historical snapshot before PR #31 merged: the `codex/fix-sonar-security-rating` branch was synced with `origin/development` in merge commit `f562ce8` and pushed. Its then-current CI status was not verified in this session.
- The 80% coverage gate, full browser/E2E acceptance, authenticated Media workflow, backup/restore test, fault-isolation/recovery scenarios, branch protection, secret rotation, tutor/group decisions, and SWE303 review evidence remain open as listed above.

## 41. GitHub integration and PR #2 Sonar follow-up (11 Nov 2026)

- PR #31 (`codex/fix-sonar-security-rating` → `development`) merged on 11 Nov as merge commit `bb3901c846e578e143be7bb6908b5d8b2fe30f34`. GitHub showed 17/17 checks passing and no conflicts before merge. Local `development` was fast-forwarded from `168d32b` to `bb3901c`; `git status --short --branch` confirmed a clean branch tracking `origin/development`.
- PR #2 remains open: `feature/norbu-match-engine` → `development`. The PR has 69 changed files and seven commits after the follow-up; GitHub reports `mergeable_state: dirty` (conflicts) with the current development branch.
- SonarCloud annotated the older PR with a browser-storage security finding in `api.js`, a cognitive-complexity finding in the BFF test, single-method Go interface naming warnings, SQL-safety warnings, and PowerShell automatic-variable warnings. The feature branch follow-up commit `5d92461576a950dd227fcad862224fda34f69dea` replaces browser access/refresh token storage with HttpOnly cookies, adds BFF cookie session/refresh/logout handling and tests, splits the high-complexity match-list test, renames the flagged interfaces, uses explicit SQL projection strings, and renames the PowerShell variables. This commit was pushed to the PR branch.
- Local verification on the PR branch passed: `go test ./...` for BFF, Player, Team, and Tournament; `node --check api.js` and `node --check app.js`; PowerShell parser check for `scripts/bff-smoke.ps1`; `docker compose -f deploy/docker-compose.yml config --quiet`; and `git diff --check`. A transient Windows Go build-temp file lock during parallel execution cleared when Player tests were rerun alone.
- After the follow-up push, the latest SonarCloud check still fails the quality gate with Security Rating B (required A); the check run returned no actionable annotations. SonarCloud's PR Summary page reported branch analysis is not included in the current plan, so further findings may require the project owner to enable branch analysis or review the check details through the project account.
- Do not merge PR #2 yet. Reconcile its conflicts with current `development`, inspect the remaining Sonar findings and rerun CI, then confirm GitHub reports no conflicts and all required checks pass. The work in `5d92461` is on the feature branch and is not part of `development` until PR #2 merges.

## 42. PR #2 development sync and conflict resolution (12 Nov 2026)

- Merged `origin/development` into `feature/norbu-match-engine` locally. Preserved the BFF HttpOnly-cookie session flow and the latest development Nginx security, Report/Media routes, registration flow, role-aware UI, and service changes. Fixture scheduling keeps the same-day venue/time conflict fallback and also retains knockout progression from finalized matches.
- Resolved all 20 conflicted paths; no conflict markers remain. Adapted Fixture fakes and PostgreSQL call sites to the merged repository interface.
- Local checks passed: `git diff --check`, `node --check` for `api.js`, `app.js`, and `role-permissions.js`, Docker Compose configuration validation, and `go test ./...` for Fixture, Player, Team, Tournament, Match, and the E2E tool module.
- The merge commit was pushed as `d88eff5`. GitHub now reports the PR branch is mergeable with development; CI is still running, so the PR is not yet merge-ready.

## 43. PR #2 SonarCloud security follow-up (12 Nov 2026)

- The SonarCloud run for `d88eff5` reported two session-cookie findings because `Secure` was set from an environment-dependent boolean. Both session and refresh cookies, including logout-clearing headers, now set `Secure: true`; BFF tests assert the flag. The cookies use HttpOnly and SameSite=Strict as before.
- Replaced four repeated E2E manager-password literals with one named constant. The initial SonarCloud rerun reported Security Rating A but 3.11% new-code duplication, just over the 3% quality-gate limit.
- Reduced repeated gRPC dependency-response handling in Player and Team clients to the shared `servicekit.FetchDependency` helper; focused Player and Team module tests pass. The revised PR still needs SonarCloud analysis to confirm the duplication gate result.

## 44. PR #2 Sonar duplication follow-up (12 Nov 2026)

- Added `servicekit.FetchDependency` to centralize gRPC dependency error mapping and required-response extraction; refactored Player→Team and Team→Tournament clients to use it.
- `go test ./...` passed serially for both Player and Team modules (`GOFLAGS=-p=1`). `git diff --check` passed. Parallel Windows Go test runs encountered temporary executable-file locks during cleanup; serial reruns completed successfully.
- SonarCloud's last observed scan before this change reported new-code Security Rating A, with duplication still slightly above the gate at 3.11%. This new commit must be pushed and analyzed before claiming the duplication gate or overall quality gate is passing.

## 45. PR #2 CI failure fixes (12 Nov 2026)

- The latest SonarCloud run on `39c7de1` reports Security Rating A and Quality Gate passed; new-code duplication is 1.47%. GitHub reports the PR branch is mergeable without conflicts.
- Reviewed the supplied Actions log. The Go security scan identified GO-2026-6859 in Go 1.26.10 (fixed in Go 1.26.11). Updated Go CI and the service build/runtime Dockerfiles to Go 1.26.11.
- The performance and audit E2E jobs target the monolith API, which Compose publishes on host port 8079; the jobs incorrectly targeted the BFF port 8080. Updated their health checks and API_BASE_URL to 8079.
- The notification E2E stack omitted the Player service even though the scenario calls it; added `player` to the services started by that job.
- Docker Compose configuration and `git diff --check` pass. Docker Desktop is unavailable in this session, and the Go 1.26.11 toolchain download stalled, so these CI changes have not yet been exercised locally. Push and rerun GitHub Actions to verify the remaining checks.

## 46. PR #2 security scan follow-up (12 Nov 2026)

- The supplied gosec output identified G706 log-injection risk in BFF request logging. Gosec still flagged client-controlled values after the URL path was removed, so the BFF now generates request IDs at its HTTP boundary and logs only response status and duration. Added a regression check that a client-provided ID is not reflected. The local gosec 2.30.0 scan now passes with zero findings, and the BFF HTTP API tests pass.
- The supplied coverage output shows all displayed module totals from `services/official` onward above 80%, but it omits earlier modules where the coverage threshold failure occurred. A local all-module coverage attempt could not complete in this environment; Docker is unavailable and the Linux shell has Go 1.22.2 pinned with `GOTOOLCHAIN=local`.
- The BFF security follow-up still needs to be committed and rescanned. The first failing module's coverage summary is still needed to diagnose the separate coverage failure.

## 47. SCRUM-69 centralized configuration consumers (13 Nov 2026)

- Added an authenticated shared configuration loader in servicekit. It reads only allowlisted non-secret settings from `service.<service-name>.<SETTING>` keys and retains environment/default values when a key is missing or Config is unavailable. Database URLs, JWT secrets, bootstrap credentials, and media storage paths remain environment-based.
- Wired the loader into the 19 domain services that consume Config and the BFF; Compose now supplies the Config address and starts Config before consumers.
- All affected Go modules compile, Docker Compose configuration validates, and `git diff --check` passes. A live central-value override still needs a runtime smoke check.

## 48. SCRUM-69 PR #32 check fixes (13 Nov 2026)

- Replaced the dynamic RPC target/method values in the circuit-breaker log with a fixed message to address gosec G706. Added the indirect Contracts dependency to the service template module for the repository tidy check.
- `go build ./...` passed for `packages/servicekit` and `services/_template`; `git diff --check` passed. The Bash security script could not run in this Windows session, so GitHub Actions must confirm the G706 fix.

## 49. SCRUM-69 coverage gate fixes (13 Nov 2026)

- Added tests for centralized configuration loading and fixture knockout progression, pending fixtures, and outbox pruning.
- `bash scripts/coverage.sh 80` passed; servicekit coverage is 83.5% and fixture coverage is 82.1%.
