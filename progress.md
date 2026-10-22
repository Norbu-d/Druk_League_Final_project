# DrukLeague — Project Progress and Handoff

**Last updated:** 2026-10-23
**Project root:** `D:\druk-league`
**Run state (23 Oct 2026):** all 20 services, the new **bff** (web API) and the monolith fallback run in Docker Compose. The web app at http://localhost:4173 now talks to the **microservices** through the bff (port 8080); the monolith moved to port 8079 as a fallback. `tools/e2e` passes **39 of 39** checks and `scripts/bff-smoke.ps1` passes **17 of 17** (section 12). Docker Desktop's disk image is on **D:** (moved by Norbu on 2026-10-19).

> **Read this first.** **11 of the 20 services have domain logic and tests** and run in Docker (Auth, User, Team, Tournament, Player, Match, Score, Standings, Fixture, Venue, Official). Registration and Dzongkhag are done in Sonam's **PR #1**, not merged yet. The other **7 are still health shells** (Notification, Audit Log, Media, Report, Search, Analytics, Configuration). The proposal and deck also promise serverless functions, OpenTelemetry and CI, none of which exist yet. Section 2 compares the code with the proposal, section 7 lists remaining work, and section 12 records what changed on 22–23 Oct. Sections 10 and 11 are kept as history.
>
> **How the team works (from 22 Oct):** feature branches merge into **`development`** through PRs; `development` is merged into `main` only when a set of work is finished and tested. Norbu now also owns Tandin's services (Auth, User, Team, Tournament, Player; Registration was done by Sonam). Sonam and Namgay build their own services. Deployment means **local Docker Compose only**, with no Kubernetes or cloud. Work not finished by submission continues afterwards, so it goes on a post-submission list rather than being cut.

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
| 20 independently deployable services | All 20 run as containers. 11 have hexagonal logic and tests (23 Oct); Registration and Dzongkhag are in PR #1; 7 remain health shells |
| API Gateway + BFF | **Done 22–23 Oct:** `services/bff` serves the web app's REST API by calling the services over gRPC. It validates the JWT once and forwards the caller's token, so every service still applies its own role checks. Published on host port 8080 and behind nginx `/api/`. There is no separate gateway: nginx and the bff fill that role. Rate limits are not done yet (11.6). Service port 50051 is not published outside Compose |
| gRPC between services | All outbound calls use `servicekit.Dial`: per-call deadline, retry on safe reads, circuit breaker, request-id and token propagation. Fixture calls Tournament/Team/Venue/Match; Official calls Match; Auth calls User; Venue, Team and Tournament call Dzongkhag; Team calls Tournament; Player calls Team; the bff calls nine services |
| Each service owns its data | Each service has its own database, and the web app now reads and writes them through the bff. The monolith keeps its own separate database, so data created through it does not appear in the web app |
| NATS JetStream event flows | Shared `packages/eventkit`: durable consumers with bounded redelivery (5) and a **dead-letter stream `DRUKLEAGUE_DLQ`**, tested against an embedded JetStream server. Match and Standings consumers use it. Fixture publishes `FixturesGenerated` from its own outbox. Notification/Audit consumers not written yet |
| OpenFaaS functions | None |
| OpenTelemetry, circuit breaker, fault isolation | Circuit breaker, deadlines, retries, request-id logging and panic recovery are in `servicekit` (unit-tested). The bff degrades a match row instead of failing the list when Team or Score is down. Team → Tournament reference check **done 22 Oct**. OpenTelemetry not started; fault-isolation scenarios not run in Docker |
| 80% unit-test coverage, CI gates | Unit + PostgreSQL integration tests for 11 services, the bff and the shared packages (sections 10 and 12). Registration and Dzongkhag tests are in PR #1. The 7 shells and the monolith have none. No CI workflow yet |
| Hexagonal architecture (cited in the literature review) | Used in every service with logic: Auth, User, Team, Tournament, Player, Match, Score, Standings, Fixture, Venue, Official, Dzongkhag, plus Registration in PR #1. Only the monolith is not |
| JWT + **bcrypt** (deck) | Services use bcrypt (cost 12) and `golang-jwt/v5`. Old PBKDF2 hashes still verify and are upgraded to bcrypt at next login. Only the monolith fallback still uses its hand-rolled code |
| Team users register and track teams | `team_manager` role in User. **22 Oct:** a team manager who creates a team becomes its `manager_id`; only that manager (or the admin) can add or edit its players. Registration checks the same `manager_id` (PR #1). No web screens for team managers yet |

### The 20 services
All modules have their own contract and Dockerfile. "e2e" means covered by `tools/e2e` against the running stack; "smoke" means covered by `scripts/bff-smoke.ps1` through the bff. Owners changed on 22 Oct: Norbu took over Tandin's services.

| Owner | Service | Status (23 Oct) |
|---|---|---|
| Norbu (was Tandin) | Auth | Done 19 Oct, tested, e2e + smoke: Login, RefreshToken (rotating), RevokeSession, lockout (5 failures / 15 min) |
| Norbu (was Tandin) | User | Done 19 Oct, tested, e2e + smoke: hexagonal, bcrypt, `VerifyCredentials`, bootstrap super admin, `team_manager` role |
| Norbu (was Tandin) | Team | **Rewritten 22 Oct, tested, e2e + smoke:** hexagonal; the tournament must exist (no more "Ghost FC") and a team manager can only join a `registration_open` one; the creating manager becomes `manager_id`; dzongkhag canonicalised; short code 2–8 letters/digits, unique per tournament |
| Norbu (was Tandin) | Tournament | **Rewritten 22 Oct, tested, e2e + smoke:** hexagonal; admin only; validated fields; dzongkhag canonicalised; `starts_at` in UTC; always starts as `registration_open`; name unique per season |
| Norbu (was Tandin) | Player | **Rewritten 22 Oct, tested, e2e:** hexagonal; the team must exist; only the admin or that team's manager can add or edit players; shirt number optional, 1–99, unique per team; family name optional |
| Sonam (was Tandin) | Registration | **Rewritten by Sonam in PR #1 (not merged):** hexagonal; `pending` / `approved` / `rejected`; checks the tournament is open and the caller manages the team; tests |
| Norbu | Match | Done 19 Oct, tested, e2e + smoke. **23 Oct:** a match books its teams, venue and official for 2 hours; overlapping bookings and double-booked officials are refused, and the error names what clashed |
| Norbu | Score | Done 19 Oct, tested, e2e + smoke: event carries `finalized_by` and `reason`; scores 0–99; outbox relay |
| Norbu | Standings | Done 19 Oct, tested, e2e + smoke: consumes `MatchFinalized` (deduplicated), team names from Team |
| Norbu | Fixture | Done 19 Oct, tested, e2e + smoke. **23 Oct:** when another tournament holds the planned slot, it tries the district's other venues, then 3 and 6 hours later the same day, and stores the slot it actually booked |
| Norbu | Venue | Done 19 Oct, tested, e2e: admin-only create, dzongkhag checked via Dzongkhag service (cached), unique name per dzongkhag |
| Sonam | Dzongkhag | Get/List, all 20 districts; tests in PR #1 |
| Sonam | Official | Done 19 Oct, tested, e2e + smoke: assigns through Match, records tournament, filter by tournament |
| Sonam | Notification, Audit Log, Media | Shells |
| Namgay | Report, Search, Analytics, Configuration | Shells |

Not one of the 20: **bff** (Norbu, 22–23 Oct, tested, smoke) is the web app's API (section 12).

---

## 3. Stack

- **Frontend:** plain HTML/CSS/JS, no build step.
- **Backend:** Go. 20 service modules under `services/`; `services/bff` (REST → gRPC for the web app); monolith fallback in `services/api-gateway`.
- **Data:** PostgreSQL 17 via `pgx/v5`; each service has its own database and runs its own migrations on start; the monolith has its own separate database.
- **Auth:** Auth issues HS256 JWTs (30-minute access token, 7-day rotating refresh token) with `golang-jwt/v5`; passwords use bcrypt. The bff and every service verify the token. The web app renews the access token through `/api/auth/refresh` when it gets a 401. The monolith fallback still uses its hand-rolled JWT/PBKDF2 code.
- **Contracts:** Protobuf + gRPC for 20 services plus `MatchFinalized`; Buf 1.73.0 lint/build/generate pass.
- **Events:** Score owns the result/outbox transaction and publishes to JetStream. Match and Standings consume the event; duplicate delivery was verified. The monolith's own outbox remains separate until gateway/API integration is changed.
- **Deployment:** Docker Compose only (PostgreSQL, NATS, nginx `web` on 4173, `bff` on 8080, monolith `api` on 8079, `db-init` job, 20 services); `deploy/build-services.ps1` builds images. No Kubernetes or cloud (decided 22 Oct).

---

## 4. File guide

- **Frontend:** root `index.html`, `styles.css`, `app.js`, `api.js` are the source of truth; `web-preview/` is a copy, do not edit it. `api.js` calls `http://<host>:8080/api` (the bff) and renews tokens on a 401.
- **bff:** `services/bff/internal/httpapi/server.go` (routes, JWT check, CORS, request ids, error mapping) and `handlers.go` (one handler per route); `cmd/bff/main.go` dials the nine services.
- **Tests and checks:** `tools/e2e` (Go, gRPC, 39 checks, runs inside the Compose network) and `scripts/bff-smoke.ps1` (PowerShell, HTTP, 17 checks, run from the host). Both leave uniquely named test rows behind.
- **Monolith (fallback):** `services/api-gateway/cmd/api/main.go` (HTTP, auth, tournaments, teams, matches, fixtures, scoring, finalisation), `admin_matches.go` (accounts, official assignment, match creation), `events.go` (outbox → JetStream), `migrations/001_init.sql` (schema + trigger blocking edits to finalised results).
- **Contracts and shared code:** `proto/` (layout: `proto/drukleague/v1/` holds events, match, score, team, tournament, user; every other service has `proto/drukleague/<name>/v1/<name>.proto`), `packages/contracts/` (generated Go), `packages/servicekit/` (migration, bearer-token check, gRPC runtime), `services/_template/` (hexagonal example), `services/<service>/` (one per service).
- **Deploy:** `deploy/docker-compose.yml`, `build-services.ps1`, `init-databases.sh`, `nginx.conf`, `.env.example` (variables: `POSTGRES_PASSWORD`, `JWT_SECRET`, `BOOTSTRAP_ADMIN_USER`, `BOOTSTRAP_ADMIN_PASSWORD`). `deploy/.env` is private: never print, copy or commit it. `.gitignore` and `.gitattributes` exist (secrets ignored; shell scripts, Go, SQL, proto, YAML, Dockerfile and nginx config forced to LF).
- **Docs (`docs/`):** `architecture-decision-mvp.md`, `service-design.md` (hexagonal layout and resilience plan), `stakeholder-analysis.md`, `wbs.md`, `risk-register.md`, `change-control.md`, `swe302-test-plan.md` (planned, not executed), `defect-log.md`, `traceability-matrix.md` (user stories and acceptance criteria).
- **Presentation:** `DrukLeague_Redesigned_v2.pptx` (19 slides, speaker notes; Tandin 1–4, Norbu 5–9, Sonam 10–13, Namgay 14–19).

### Monolith behaviour (fallback only since 22 Oct)
- Bearer-token auth; the Super Admin creates match officials and viewers; team and tournament writes are admin-only.
- Match creation rejects same-kickoff team/venue clashes; round-robin generation uses a per-tournament advisory lock and does not duplicate.
- Only the assigned official can update or finalise a score. In the **monolith**, finalisation writes the result, both standings rows and the outbox event in one transaction; a trigger rejects later edits.
- On startup the API creates a super-admin only if none exists, so changing the bootstrap password in `.env` later does not change an existing admin.

### Split-service finalisation consistency

The monolith's single transaction cannot span the Score and Standings databases. In the split architecture, Score commits the final result and its outbox event atomically. Standings updates by consuming `MatchFinalized`, so standings are eventually consistent with the final result. The consumer deduplicates by `event_id`, which Score includes in the event payload. Match retains `scheduled`/`completed` status and marks a match completed when it consumes `MatchFinalized`. These flows and duplicate handling were verified on 14 Oct 2026.

Web API routes. The bff serves them with the same JSON shapes the monolith used, so the frontend did not change; the README lists request bodies and error codes.

| Route | Purpose |
|---|---|
| `POST /api/auth/login` | log in; returns `token`, `refresh_token`, `user` |
| `POST /api/auth/refresh`, `POST /api/auth/logout` | renew the access token; revoke the session (bff only) |
| `/api/users` | list/create accounts (Super Admin) |
| `/api/tournaments`, `/api/teams` | create/list |
| `/api/matches` | create/list; `/{id}/official`, `/{id}/live`, `/{id}/finalize` (the bff saves the submitted score, then finalises it) |
| `/api/tournaments/{id}/fixtures` | generate round-robin fixtures (the bff starts them on the tournament's own start date) |
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

### Verified 22–23 Oct 2026 (automated)
- `tools/e2e`: **39 of 39** checks pass (the 29 from 19 Oct plus 10 for Tournament, Team ownership and Player rules).
- `scripts/bff-smoke.ps1`: **17 of 17** HTTP checks pass, run twice back to back: login, refresh, logout, tournaments, teams, fixture generation (including a second same-day tournament in the same dzongkhag that must be rescheduled), official assignment, live score, finalise with the submitted score, standings, and nginx proxying.
- The new unique-index migrations (team short code, tournament name, shirt number) applied cleanly to the existing service databases, so there were no duplicates.
- Still not exercised: NATS outage/reconnect, dead-letter replay, fault-isolation scenarios.

### Docker rules
- **Docker Desktop's disk image was moved to D: on 2026-10-19** (Settings > Resources > Advanced). On 19 Oct, D: had about 122 GB free and C: about 23 GB. Keep at least 10 GB free on both drives. Earlier, a full C: drive (Docker's `docker_data.vhdx` was 25.8 GB) caused the hangs and HTTP 500s.
- Start with `docker compose up -d --no-build`. Build one service at a time: `$env:COMPOSE_PARALLEL_LIMIT = 1`, then `docker compose build <service>` (repeat the `$env:` line in each new PowerShell window).
- After a restart the engine may return HTTP 500 for a minute or two. Wait 60–90 seconds, or run `wsl --shutdown` and reopen Docker Desktop.
- Pruning frees space inside Docker's disk file; the file itself only shrinks after compacting (`diskpart`, `compact vdisk`, with Docker and WSL stopped).
- **Never run** `docker volume prune`, `docker system prune --volumes` or `docker compose down -v`. Keep the volumes `drukleague-postgres-data` (PostgreSQL 16, preserved), `drukleague-postgres17-data` (used by Compose) and `deploy_nats-data`.
- Minimum machine for teammates: about 10 GB free disk and 8 GB RAM.
- **After pulling the 22–23 Oct changes**, build the new bff image once (`docker compose build bff`), rebuild team, tournament, player, match and fixture, then `docker compose up -d --no-build`. If port 8080 is busy, the old `api` container is still holding it; `docker compose up -d --no-build api` moves it to 8079.
- From Git Bash, prefix `docker` commands that contain Linux paths with `MSYS_NO_PATHCONV=1`, or use PowerShell; Git Bash rewrites `/var/lib/...` into a Windows path.

---

## 6. Open decisions

1. **Password hashing:** *resolved in code 19 Oct:* the services use bcrypt, matching the deck. The monolith still uses PBKDF2 until Part 5 retires it.
2. **Team-user role:** role name `team_manager` in code. Team and Player ownership done 22 Oct; Registration ownership in PR #1. Still align the proposal and deck wording.
3. **Serverless functions:** *deployment decided 22 Oct:* Docker Compose only, no Kubernetes or cloud. No function exists yet. If one is not built by submission (faasd, or a function-style container in Compose), list it as post-submission work in the report.
4. **Risk R1:** the deck rates "20 services in one semester" High/High; the proposal says Medium. Make them match.
5. **Finalised-result correction:** admin correction with a reason + audit entry (user stories) vs append-only correction events (architecture note). **22 Oct recommendation:** admin-only void-and-replace with an audit record (section 11.4).
6. **Sprint 4/5 dependency:** *no longer blocking:* Venue exists and Fixture's clash handling is done (23 Oct). Record the reorder in the sprint evidence.
7. **NATS resilience:** bounded redelivery and dead-lettering are implemented in `eventkit` and pass against an embedded JetStream server. Still test NATS outage/reconnect in Docker, and write the Notification/Audit consumers.
8. **Hexagonal and resilience wording:** the proposal cites hexagonal architecture but never says it will be applied. Add one methodology sentence and a resilience objective, then update the deck's architecture and testing slides, the risk table and the speaker notes.
9. **Cross-service integrity:** *resolved 22 Oct:* Team checks the tournament synchronously through Tournament (resilient client), so a nonexistent `tournament_id` is refused. Record the finding and the fix in the risk register. Moving the link to Registration (item 10) is still the cleaner long-term model.
10. **Registration model (proposed 22 Oct):** Registration links Team to Tournament with a status; Team becomes a persistent entity; Fixture reads approved registrations. Needs agreement with Tandin (section 11.3).
11. **Service identity and user → official mapping (proposed 22 Oct):** a distinct service claim instead of a borrowed `super_admin`, and an explicit mapping from user ID to official ID. Needs agreement with Tandin and Sonam (section 11.2).
12. **Scope after submission:** *changed 22 Oct:* instead of cutting OpenTelemetry, the five shells and asymmetric JWT signing, anything unfinished at submission goes on a post-submission list (work continues in the vacation). The group should confirm, and the report should say which items are post-submission.

Decided: build toward all 20 services (7 Oct 2026); Dzongkhag Admin role dropped (7 Oct 2026); integrate on `development` through PRs, and Norbu takes over Tandin's services (22 Oct 2026); Docker Compose is the only deployment target (22 Oct 2026).

---

## 7. Remaining work (priority order)

Updated 23 Oct. Owners in brackets.

**Now (integration)**
1. **Merge into `development`** (Norbu): push `feature/norbu-match-engine` and open a PR into `development`. It changes shared files (`deploy/docker-compose.yml`, `deploy/nginx.conf`, `deploy/build-services.ps1`, `api.js`, `README.md`), so tell the group before merging.
2. **Sonam's PR #1** (Sonam, review by Norbu): change its base to `development` and merge. It works with the new Team: Registration's ownership check reads the `manager_id` that Team now saves.
3. **Teammates' branches** (Namgay, Sonam): `feature/namgay-insights-testing` and `development` were 2 commits behind `main` on 22 Oct; merge `origin/development` before starting. `feature/tandin-core-services` is no longer used.

**Services**
4. **The 7 shells** (Sonam: Notification, Audit Log, Media; Namgay: Report, Search, Analytics, Configuration). Copy the hexagonal pattern from Venue or Team; Notification and Audit Log should consume `MatchFinalized` through `eventkit`. Audit Log is the most useful first (11.6).
5. **Result correction** (group decision first, open decision 5): admin-only void-and-replace with an audit record. A Standings rebuild (replay final results from Score) would make it simpler than reversing standings rows.
6. **Registration model** (open decision 10): move the tournament link from Team to Registration and have Fixture read approved registrations. Only after PR #1 is merged.

**Testing and CI (SWE302)**
7. **Coverage:** every tested package except Fixture's PostgreSQL adapter (78.9%) is at or above 80% (section 12). Add tests for the shells and Registration/Dzongkhag as they land. `scripts/coverage.sh` still prints FAIL on Windows Git Bash; fix it before using it as a gate.
8. **CI:** one GitHub Actions workflow with `go vet`, `go test -cover` per module (fail under 80%), `govulncheck`, `gosec`; later Trivy, `buf breaking` (needs a baseline), k6 against the bff, Playwright.
9. **Fault isolation in Docker:** stop Team or Score and show the bff still lists matches (degraded rows); stop NATS and show Score's outbox catches up afterwards.

**Frontend**
10. **Screens missing:** players, registrations, venues, officials and team-manager views have APIs but no pages; the dashboard still mixes sample data with API data; loading, error and empty states.
11. **Demo data:** the web app shows every tournament in the service databases, including leftover e2e and smoke-test rows. Before a demo, use a fresh database or create clean data through the web app.

**Other**
12. **Serverless function** and **OpenTelemetry:** not started; post-submission if not done (open decisions 3 and 12).
13. **Secrets:** rotate `deploy/.env` values before the repository is shared more widely. `JWT_SECRET` can change on its own (then recreate the services); `POSTGRES_PASSWORD` and `BOOTSTRAP_ADMIN_PASSWORD` need a coordinated change. Bind DB/API ports to loopback. Confirm the GitHub repository is private and `main` is protected.
14. **SWE303 evidence:** sprint plans, reviews and retrospectives; add the 22–23 Oct defects (section 12) to `docs/defect-log.md`; map e2e and smoke checks to user stories in `docs/traceability-matrix.md`; update the risk register (cross-service integrity, R1 rating).

**Done (kept for the record):** Docker disk moved to D: (19 Oct); GitHub repository created and pushed; Parts 2–4 extracted and containerised (14 and 19 Oct); Part 5 bff and frontend cutover (22–23 Oct); Part 7 Team → Tournament check (22 Oct); finalise uses the submitted score (22 Oct); Match overlap and official clash checks, Fixture rescheduling (23 Oct).

**Sprint mapping (proposal's 8 sprints):** S1 foundation/CI/contracts · S2 identity + gateway + BFF · S3 teams/tournaments/registration · S4 competition (match → standings) · S5 venues/officials/notifications · S6 supporting services · S7 integration, events, serverless, observability, security/performance tests · S8 QA, coverage, defects, demo.

---

## 8. Continue in a new Claude chat

**In Claude Code (VS Code extension), the way this was built since 22 Oct:** Claude can read the project, write code, and run `go test`, Docker builds, `tools/e2e` and `scripts/bff-smoke.ps1` itself. Open the project folder and send:

> DrukLeague is a Go microservices project. Read progress.md (start with "Read this first", sections 2, 7 and 12) and README.md. Never print, copy or commit deploy/.env; scripts may read the admin password from it without printing it.
>
> **Rules:** follow the Docker rules in progress.md section 5 (one build at a time, never `docker volume prune` or `down -v`). Work on my feature branch and open PRs into `development`. Copy the hexagonal pattern of an existing service (Venue or Team). Run the unit and PostgreSQL tests, rebuild the changed services, and run `tools/e2e` and `scripts/bff-smoke.ps1` before saying something works.
>
> **Task:** <the item from progress.md section 7>.

**In a web chat that cannot see the project:** give it `progress.md` and `README.md`, paste the files for the service you are working on (and one finished service such as `services/venue` as the pattern), and ask for full-file code plus PowerShell commands. Run them yourself and paste back only the output. Tell it not to claim anything works until you have shown it the output.

---

## 9. Folder structure

As of 12 Oct 2026, taken from the files Git is tracking. `deploy/.env` exists locally but is private and not committed. **Added since:** `packages/eventkit/` and `packages/servicekit/{core,dzongkhags,testdb}/` (19 Oct), `tools/e2e/` (19 Oct), `services/bff/` and `scripts/bff-smoke.ps1` (22–23 Oct). Every service with logic now has the full `internal/{domain,app,ports,adapters}` layout and `*_test.go` files.

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

**Target layout (bff exists since 22 Oct; the rest is still to do):**

```text
services/
├── bff/                  # done: web API over the services
├── api-gateway/          # monolith fallback; could be retired once the web app no longer needs it
└── <service>/            # hexagonal: cmd/, internal/{domain,app,ports,adapters}/, migrations/, Dockerfile, tests
functions/                # serverless function(s), e.g. notify-on-match-finalized (not started)
.github/workflows/        # CI: lint, test, coverage gate, scans (not started)
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

---

## 11. Design review — 22 Oct 2026

**Status (23 Oct): partly implemented; kept as written for the record.** Section 7 is now the current work list. Done since the review: 11.1 steps 4 (bff and frontend cutover, fuller than the minimum) and 6 (Team, Tournament and Player tests), and from step 3 the finalise fix and the official-overlap check. In 11.2, Team ownership. In 11.4, finalise uses the submitted score (through the bff), matches have a 2-hour duration with official clash checks, and byes and UTC storage were already handled in Fixture and Match. In 11.6, the token check: the web app keeps both tokens in `sessionStorage` (a known limitation). In 11.7, the decision is Docker Compose only, and the tutor will not be asked. Not done: step 1 (tag), the registration model, result correction, CI, the rest of 11.2, 11.5 and 11.6.

The findings came from reading `README.md` and `progress.md`, not from re-reading the code, so check each one against the code before acting. Items marked *group decision* touch services owned by other members.

### 11.1 Proposed order of work

1. **Tag the known-good state.** Commit and create a git tag on the 19 Oct state (29/29 e2e checks) so there is something to return to.
2. **Write three short design notes (about half a page each) and agree them with the group:** the registration model (11.3), the result correction path (11.4), and service identity plus the user → official mapping (11.2). Tandin owns Team, Registration and Auth; Sonam owns Official.
3. **Registration and match fixes** (11.3, 11.4): move the tournament link to Registration with a status; Fixture reads approved registrations; finalise carries the score or a version and rejects a mismatch; add an official-overlap check in Match. Start with Match, Score and Fixture (Norbu's services) and hand Tandin the Registration contract change.
4. **Minimum gateway and frontend cutover.** Not the full BFF: a small Go gateway that validates the JWT and exposes REST for one flow end to end (login, list tournaments, scoring and finalisation). Switch the frontend over flow by flow and keep the monolith as a fallback. This is what makes the microservices visible in a demo.
5. **CI and coverage gate** (11.5, 11.6): fix `scripts/coverage.sh` first, then one GitHub Actions workflow running `go test -cover` per module and failing under 80%.
6. **Tests for Team, Tournament, Player and Registration**, after step 3 so they cover the new registration design.
7. **If time remains:** Notification and Audit Log consuming `MatchFinalized`; one or two OpenFaaS/faasd functions (11.7).

**Proposed to cut, or to document as a known limitation** (*group decision*, because 7 Oct 2026 decided to build all 20 services): OpenTelemetry, the other five shells (Media, Report, Search, Analytics, Configuration), and asymmetric JWT signing.

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
- **Dead-letter stream is write-only.** Add a way to list and replay `DRUKLEAGUE_DLQ` (admin tool or runbook).
- **Retention.** Clean up published outbox rows and set retention limits on the JetStream streams.
- **Double-submit protection.** Idempotency keys or natural-key unique constraints on create calls (register team, finalise).
- **Test data.** The e2e run leaves rows behind. Prefix test data and add a cleanup path, or run e2e against a throwaway database.
- **Migration risk.** User `002_roles.sql` fails if two usernames differ only by case. Add a pre-check or dedupe step.

### 11.6 Frontend, edge and report evidence

- **nginx:** rate limits on login and public viewer reads, CORS, basic security headers.
- **Token storage.** Check where the plain-JS frontend keeps the JWT. Prefer the access token in memory and the refresh token in an httpOnly cookie if the gateway can set it; otherwise document it as a limitation.
- **`web-preview/`** is a copy that must not be edited. Delete it or generate it by script so it cannot drift.
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

## 12. Session log — 22–23 Oct 2026 (Norbu, with Claude Code)

All work is on `feature/norbu-match-engine` (commits `56aa481`, `bb49acf`, `f90d524`, `837a4eb`), to be merged into `development` through a PR.

### What changed

**Team, Tournament, Player (taken over from Tandin, 22 Oct).** Rewritten in the hexagonal layout with domain rules, ownership and reference checks; see the table in section 2. New migrations, each with a unique index that is skipped with a notice (instead of failing the service) if old rows already clash: `team/002_manager_and_unique_code.sql`, `tournament/002_unique_name.sql`, `player/002_unique_shirt.sql`. No contract changes.

**bff (new, 22–23 Oct).** `services/bff` serves the monolith's REST routes and JSON shapes by calling Auth, User, Tournament, Team, Match, Score, Standings, Fixture and Official.
- It validates the JWT once, forwards the caller's token, and gives every request an id that reaches the service logs.
- `GET /api/matches` joins Match, Team names and Score. A Team or Score outage degrades that row instead of failing the list.
- Finalise saves the submitted score, then finalises it. Fixture generation starts on the tournament's own start date. One-letter short codes from the web form are widened (e.g. "Paro" becomes PAR).
- New `/api/auth/refresh` and `/api/auth/logout`. `api.js` renews the token on a 401 and revokes the session on sign-out.
- Compose: `bff` on host port 8080 and behind nginx `/api/`; the monolith `api` moved to 8079 as a fallback. To switch back, swap the two ports and point nginx at `api:8080`.

**Match (23 Oct).** A match occupies its teams, venue and official for two hours (`domain.SlotLength`). Creating a match within two hours of another booking of the same team or venue (venue names compared case-insensitively) is refused, as is assigning an official who has another unfinished match within two hours. Errors name what clashed and when. All booking changes are serialised by one advisory lock. Migration `003_schedule_indexes.sql`.

**Fixture (23 Oct).** When Match reports a clash, Fixture tries the district's other venues at the planned time, then 3 and 6 hours later the same day, and stores the slot actually booked (`LinkMatch` now saves venue and kickoff). A tournament plays at most once a day, so these moves cannot hit its own schedule. If the whole day is taken, it reports the date, round and last clash, and generation can be re-run later.

**Tests and tooling.** `tools/e2e` grew from 29 to 39 checks, and two checks that only passed on a clean database were fixed. New `scripts/bff-smoke.ps1` (17 HTTP checks).

### Test coverage (23 Oct 2026, `go test -cover`, PostgreSQL integration tests enabled)

| Module | domain | app | gRPC server | gRPC clients | NATS | PostgreSQL |
|---|---|---|---|---|---|---|
| Team | 100% | 100% | 96.3% | 100% | — | 92.0% |
| Tournament | 100% | 100% | 96.7% | — | — | 93.9% |
| Player | 100% | 100% | 97.2% | 100% | — | 93.3% |
| Match | 100% | 100% | 98.3% | 100% | 100% | 83.5% |
| Fixture | 100% | 94.9% | 100% | 100% | 100% | **78.9%** (below 80%) |
| bff | `httpapi` 90.4% | | | | | |

Score, Standings, Venue, Official, User and Auth are unchanged since section 10.

### Defects found and fixed (for `docs/defect-log.md`)

| # | Defect | Fix |
|---|---|---|
| 1 | Team accepted a nonexistent `tournament_id` ("Ghost FC") | Team checks Tournament before saving |
| 2 | Registration's ownership check (PR #1) could never pass: Team never stored `manager_id` | Team records the creating team manager |
| 3 | Finalise ignored the submitted score and finalised the last live score | The bff saves the submitted score first |
| 4 | Match only refused an identical kickoff; matches 30 minutes apart at one venue were accepted; no official overlap check | Two-hour slots for teams, venues and officials |
| 5 | Fixture generation failed outright when another tournament held the planned venue and time | Fixture tries other venues and later slots the same day |
| 6 | The web app session broke after 30 minutes (Auth's access token TTL) because the frontend never renewed it | Refresh route in the bff, renewal in `api.js` |
| 7 | The tournament start date from the web form was ignored when generating fixtures | The bff passes it to Fixture |
| 8 | e2e: the venue check failed on every run after the first, and runs within the same hour clashed | Check the venue's district; minute-precision start |

### Known limitations (state them in the report if not fixed)

- Both tokens are in `sessionStorage`, readable by page scripts. An httpOnly refresh cookie would be safer.
- `GET /api/matches` makes one Score call and up to two Team calls per match. Fine at demo scale; it needs a batch read before it is used with hundreds of matches.
- One advisory lock serialises all match bookings. Correct, but it would become a bottleneck under heavy load.
- Fixture only reschedules within the same day.
- The monolith fallback has its own database, so the two never show the same data.
