# DrukLeague

DrukLeague is a tournament operations platform for Bhutan's inter-dzongkhag competitions. It replaces phone calls, paper score sheets and messaging apps with one place to create tournaments, register teams, generate fixtures, submit live scores, finalise results and publish standings.

The repository holds:

1. **A plain HTML/CSS/JS frontend** that talks to a REST API.
2. **A backend-for-frontend** (`services/bff`) that serves that REST API by calling the microservices. It validates the JWT once and forwards the caller's token, so each service still applies its own role checks.
3. **20 Go microservices** (gRPC, PostgreSQL database per service, NATS JetStream events). 13 have real logic; 7 are still health-only shells.
4. **The original MVP monolith** (`services/api-gateway`), kept as a fallback on port 8079. It has its own database, so data created through one is not visible through the other.

For detailed status and the handoff notes, see [progress.md](progress.md); planned design changes from the 22 Oct review are in its section 11. For the target architecture, see [servicedesign.md](servicedesign.md).

## Run locally with Docker Compose

Needs Docker Desktop, about 10 GB free disk and 8 GB RAM. From the `deploy` directory:

```powershell
Copy-Item .env.example .env
# Edit .env: set POSTGRES_PASSWORD, JWT_SECRET (32+ random bytes),
# BOOTSTRAP_ADMIN_USER and BOOTSTRAP_ADMIN_PASSWORD. Never commit .env.
.\build-services.ps1
docker compose up -d
```

- Web app: http://localhost:4173
- Web API (bff → microservices): http://localhost:8080
- Monolith fallback API: http://localhost:8079. To switch the web app back to it, swap the two ports in `docker-compose.yml` and point nginx's `/api/` at `api:8080`.
- Log in with the `BOOTSTRAP_ADMIN_USER` / `BOOTSTRAP_ADMIN_PASSWORD` you set. The admin is created only on first start, so changing the password in `.env` later does not change an existing admin.

The microservices listen on gRPC port 50051 inside the Compose network only; it is not published to the host.

**Docker tips:** on a low-resource machine, build one service at a time (`$env:COMPOSE_PARALLEL_LIMIT = 1`, then `docker compose build <service>`) and start with `docker compose up -d --no-build`. Never run `docker volume prune`, `docker system prune --volumes` or `docker compose down -v`; they delete the database and NATS volumes.

## Microservice status

| Service | Status |
|---|---|
| Auth | Login, rotating refresh tokens, session revoke, lockout after 5 failures (15 min) |
| User | bcrypt passwords, `VerifyCredentials`, roles (`super_admin`, `match_official`, `team_manager`, viewer), bootstrap admin |
| Match | Scheduling with team/venue clash checks, official assignment, marks matches completed on `MatchFinalized` |
| Score | Live scores (0–99), one-time finalisation by the assigned official, transactional outbox → JetStream |
| Standings | Consumes `MatchFinalized` (deduplicated by `event_id`), win/draw/loss points, team names from Team |
| Fixture | Round-robin and double round-robin, venue rotation, creates matches in Match, `FixturesGenerated` outbox |
| Venue | Admin-only create, dzongkhag checked through the Dzongkhag service, unique name per dzongkhag |
| Official | Assigns through Match, filter by tournament |
| Dzongkhag | Reference data for the 20 dzongkhags |
| Team, Tournament, Player, Registration | Basic create/get/list (Player update, Registration review). Not yet hexagonal; Team does not check that `tournament_id` exists |
| Notification, Audit Log, Media, Report, Search, Analytics, Configuration | Shells (gRPC health only) |

Shared packages:

- `packages/servicekit`: JWT auth (`golang-jwt/v5`), gRPC client with per-call deadlines, retries on safe reads and a circuit breaker; request-id logging, panic recovery, error mapping, migrations.
- `packages/eventkit`: JetStream streams, durable consumers with bounded redelivery, dead-letter stream `DRUKLEAGUE_DLQ`, de-duplicated publisher.
- `packages/contracts`: generated Go code from `proto/` (Buf).

Services with real logic follow a hexagonal layout: `internal/{domain,app,ports,adapters}`, with `cmd/service/main.go` doing wiring only.

### Not built yet

- Frontend pages beyond the original MVP flows (players, registrations, venues and officials have APIs but no screens).
- Notification and Audit Log event consumers.
- OpenFaaS/faasd serverless functions.
- OpenTelemetry tracing.
- The 7 shell services above.
- CI (GitHub Actions), and tests for the 12 modules that have none.

### Known design gaps and planned changes

Found in a design review on 22 Oct 2026. **Planned, not implemented.** Details and the proposed order of work are in [progress.md](progress.md) section 11.

- **Auth:** shared HS256 secret (any verifying service can mint tokens); no distinct service identity; no explicit user → official mapping; no password change, admin reset or deactivate; lockout can be abused to lock out an account; `manager_id` ownership not enforced.
- **Registration:** the tournament link should move from Team to Registration (with `pending`/`approved`/`rejected`/`withdrawn` status) and Fixture should read approved registrations; deadline, capacity and roster-lock rules are undefined.
- **Matches and fixtures:** no correction path for a wrongly finalised result; only `scheduled`/`completed` states; withdrawals and late teams not handled; tie-breakers undefined. *Done 23 Oct:* finalise through the bff uses the submitted score; matches occupy a two-hour slot and Match refuses overlapping bookings of a team, venue or official; Fixture moves a fixture to another venue or a later kickoff the same day when its planned slot is taken; byes and UTC storage were already handled.
- **Operations:** Standings cannot be rebuilt, the dead-letter stream cannot be inspected or replayed, no backup script, no outbox or stream retention, no idempotency on create calls, no readiness checks in Compose.
- **Edge and frontend:** rate limiting, CORS and security headers at nginx; JWT storage in the frontend; the `web-preview/` duplicate copy.
- **CI and evidence:** `buf breaking`, govulncheck, gosec, Trivy and a coverage gate in CI; a k6 performance test; requirement → test traceability.
- **Deployment:** Compose stays. Proposed serverless runtime is faasd (OpenFaaS on one machine), not Kubernetes; confirm with the module tutor.

## Testing

Unit and PostgreSQL integration tests exist for Auth, User, Team, Tournament, Player, Match, Score, Standings, Fixture, Venue, Official and the shared packages. Integration tests are skipped unless `TEST_DATABASE_URL` is set:

```powershell
docker run -d --rm --name drukleague-testdb -e POSTGRES_PASSWORD=test -p 127.0.0.1:55432:5432 --tmpfs /var/lib/postgresql/data postgres:17-alpine
$env:TEST_DATABASE_URL = "postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable"
cd services\match; go test -cover ./...
```

End-to-end smoke test against the running Compose stack (39 checks across login, roles, team ownership, player rosters, fixtures, scoring, finalisation and event consumers):

```powershell
docker run --rm --network deploy_default -v D:/druk-league:/src -w /src/tools/e2e -e E2E_ADMIN_PASSWORD=<your BOOTSTRAP_ADMIN_PASSWORD> drukleague-service-builder:local go run .
```

HTTP smoke test of the web API (the bff), driving the same routes as the browser, from login through fixtures (including a same-day tournament that must be rescheduled), scoring and standings (17 checks). It reads the admin password from `deploy/.env` without printing it:

```powershell
.\scripts\bff-smoke.ps1
```

Each run leaves uniquely named test rows behind, because the services have no delete RPCs.

## Web API

Served by the bff on port 8080 (and through nginx at `/api/` on port 4173), with the same routes and JSON shapes as the monolith so the frontend did not change. All `/api/*` routes except login and refresh require `Authorization: Bearer <token>`.

- `POST /api/auth/login`: `{ "username", "password" }` → `{ "token", "refresh_token", "user" }`
- `POST /api/auth/refresh`: `{ "refresh_token" }` → new `token` and rotated `refresh_token` (access tokens last 30 minutes; the frontend renews automatically)
- `POST /api/auth/logout`: revokes the session
- `GET|POST /api/users`: super admin lists accounts or creates one `{ "username", "password", "role" }`
- `GET|POST /api/tournaments`
- `GET|POST /api/teams`
- `GET|POST /api/matches`
- `PATCH /api/matches/{id}/official`: `{ "official_id": "..." }` (admin assignment)
- `PATCH /api/matches/{id}/live`: `{ "home": 1, "away": 0 }`
- `POST /api/matches/{id}/finalize`: `{ "home": 2, "away": 1 }` saves and finalises the submitted score (an empty body finalises the last live score). The monolith ignores the body.
- `POST /api/tournaments/{id}/fixtures`
- `GET /api/standings?tournament_id={id}`
- `GET /healthz`

Errors are `{ "error": "message" }`; service errors map to HTTP as InvalidArgument 400, Unauthenticated 401, PermissionDenied 403, NotFound 404, AlreadyExists/FailedPrecondition 409, Unavailable 503. In `GET /api/matches`, a Team or Score outage degrades the affected rows ("Unknown team", no score) instead of failing the list.

The monolith still uses its own PBKDF2 password hashing and hand-rolled JWT code; the microservices use bcrypt and `golang-jwt/v5`. Finalisation writes the result, both standings rows and an outbox event in one transaction, and a PostgreSQL trigger blocks later edits.

To run the monolith without Docker, set `DATABASE_URL`, `JWT_SECRET` and `NATS_URL`, then run `go run ./cmd/api` from `services/api-gateway`.

## Repository layout

```text
proto/              Protobuf contracts (Buf)
packages/           contracts (generated), servicekit, eventkit
services/           20 microservices, bff (web API over the microservices), api-gateway (monolith fallback), _template, shared builder/runtime images
deploy/             docker-compose.yml, nginx.conf, .env.example, build-services.ps1
tools/e2e/          end-to-end smoke test
docs/               project management and test planning documents
index.html, app.js, api.js, styles.css   frontend
```