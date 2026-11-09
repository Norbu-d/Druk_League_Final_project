# DrukLeague

DrukLeague is a tournament operations platform for Bhutan's inter-dzongkhag competitions. It replaces phone calls, paper score sheets and messaging apps with one place to create tournaments, register teams, generate fixtures, submit live scores, finalise results and publish standings.

The repository holds two things that are not yet connected:

1. **An MVP monolith** (`services/api-gateway`) with a plain HTML/CSS/JS frontend. Most browser API calls still go here; report creation and downloads are proxied to the Report service.
2. **20 Go microservices** (gRPC, PostgreSQL database per service, NATS JetStream events) that are replacing the monolith. Several modules are still incomplete or health-only; see the service table and [progress.md](progress.md) for current boundaries.

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
- Monolith API: http://localhost:8080
- Log in with the `BOOTSTRAP_ADMIN_USER` / `BOOTSTRAP_ADMIN_PASSWORD` you set. The admin is created only on first start, so changing the password in `.env` later does not change an existing admin.

The gRPC microservices listen on port 50051 inside the Compose network only. Report downloads and media upload/retrieval use authenticated same-origin HTTP routes through the web container on port 4173. Media bytes live in the persistent `media-data` volume; PostgreSQL stores only metadata.

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
| Notification | Consumes `MatchFinalized` and stores idempotent in-app result notices for participating team managers |
| Report | Generates persistent tournament standings and fixture reports as CSV/JSON; authenticated same-origin downloads |
| Audit Log | Persistent PostgreSQL audit entries, admin-only paginated reads, authenticated recording RPC, and durable MatchFinalized/admin-action consumers |
| Analytics | Persists finalized-match facts idempotently and serves filtered tournament, participating-team and completed-match dashboard counts; runtime RPC acceptance remains pending |
| Configuration | Authenticated read and super-admin-only write gRPC endpoints backed by PostgreSQL; runtime RPC acceptance remains pending |
| Media | Authenticated same-origin upload, match media listing, retrieval and owner/admin deletion; metadata in PostgreSQL and binary assets in the persistent Compose `media-data` volume. Supports JPEG/PNG/WebP photos and MP4 highlights up to 10 MiB. |

Shared packages:

- `packages/servicekit`: JWT auth (`golang-jwt/v5`), gRPC client with per-call deadlines, retries on safe reads and a circuit breaker; request-id logging, panic recovery, error mapping, migrations.
- `packages/eventkit`: JetStream streams, durable consumers with bounded redelivery, dead-letter stream `DRUKLEAGUE_DLQ`, de-duplicated publisher.
- `packages/contracts`: generated Go code from `proto/` (Buf).

### Inspecting and replaying dead-letter events

The `tools/event-admin` command inspects recent entries in `DRUKLEAGUE_DLQ` and replays an entry to its recorded original subject. Configure `NATS_URL` if NATS is not at `nats://localhost:4222`:

```powershell
$env:NATS_URL = "nats://localhost:4222"
go run ./tools/event-admin inspect --limit 50
go run ./tools/event-admin replay --dry-run 123
go run ./tools/event-admin replay 123
```

Replay retains the original dead-letter entry for audit. The DLQ stream keeps entries for 30 days.

### Rebuilding standings

With the Compose services already running, rebuild the standings projection from the retained `MatchFinalized` event history:

```powershell
docker compose -f deploy/docker-compose.yml build standings
docker compose -f deploy/docker-compose.yml run --rm --no-deps --entrypoint /app/rebuild standings
```

The command refuses to rebuild if the event stream is empty or its first sequence is no longer present. It pauses the live standings consumer while replacing its projection and processed-event IDs in one database transaction.

Services with real logic follow a hexagonal layout: `internal/{domain,app,ports,adapters}`, with `cmd/service/main.go` doing wiring only.

### Not built yet

- API gateway and BFF. nginx still routes `/api/` to the monolith, so the frontend does not use the microservices.
- OpenFaaS/faasd serverless functions.
- OpenTelemetry tracing.
- Team → Tournament reference check.
- Search indexing/data synchronization beyond the search query endpoint.
- Service-wide runtime acceptance for Analytics and Configuration; authenticated browser acceptance for Media.
- Tests for the remaining modules without `_test.go` files, plus browser E2E and Docker fault-injection scenarios.

Local Compose publishes the web app and monolith API on loopback (`127.0.0.1`), rather than all network interfaces.

### Known design gaps and planned changes

Found in a design review on 22 Oct 2026. **Planned, not implemented.** Details and the proposed order of work are in [progress.md](progress.md) section 11.

- **Auth:** shared HS256 secret (any verifying service can mint tokens); no distinct service identity; no explicit user → official mapping; no password change, admin reset or deactivate; lockout can be abused to lock out an account.
- **Registration:** the tournament link should move from Team to Registration (with `pending`/`approved`/`rejected`/`withdrawn` status) and Fixture should read approved registrations; deadline, capacity and roster-lock rules are undefined.
- **Matches and fixtures:** no correction path for a wrongly finalised result; finalise uses the last saved live score, not a submitted one; only `scheduled`/`completed` states; byes, withdrawals and late teams not handled; no official-overlap check; tie-breakers undefined; `starts_at` should be stored in UTC (Bhutan is UTC+6).
- **Operations:** Standings rebuild and dead-letter inspection/replay tools are implemented; run-time recovery scenarios, database backup/restore, create-call idempotency and some readiness checks still need acceptance.
- **Edge and frontend:** Nginx rate limits and baseline security headers are configured; cross-origin access is limited to the local web origin. The API JWT is stored in an HttpOnly, SameSite=Strict cookie; a refresh-token flow is not implemented.
- **CI and evidence:** GitHub Actions now runs Go formatting, vet, build, tests and an 80% coverage gate for modules with tests, plus Buf compatibility checks, govulncheck, gosec, Trivy image scans, and a k6 login/public-standings performance test. Test coverage for modules without tests remains outstanding.
- **Deployment:** Compose stays. Proposed serverless runtime is faasd (OpenFaaS on one machine), not Kubernetes; confirm with the module tutor.

## Testing

Unit and PostgreSQL integration tests exist for Auth, User, Team, Match, Score, Standings, Fixture, Venue, Official, Notification, Report, Media, Analytics and shared packages. Integration tests are skipped unless `TEST_DATABASE_URL` is set:

```powershell
docker run -d --rm --name drukleague-testdb -e POSTGRES_PASSWORD=test -p 127.0.0.1:55432:5432 --tmpfs /var/lib/postgresql/data postgres:17-alpine
$env:TEST_DATABASE_URL = "postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable"
cd services\match; go test -cover ./...
```

End-to-end smoke test against the running Compose stack (29 checks across login, roles, fixtures, scoring, finalisation and event consumers):

```powershell
docker run --rm --network deploy_default -v D:/druk-league:/src -w /src/tools/e2e -e E2E_ADMIN_PASSWORD=<your BOOTSTRAP_ADMIN_PASSWORD> drukleague-service-builder:local go run .
```

Each run leaves uniquely named test rows behind, because the services have no delete RPCs.

## Monolith API

This is the monolith API currently used by the frontend. Login establishes an HttpOnly session cookie; same-origin browser requests send it automatically. Service-to-service and command-line clients use their configured authentication headers.

- `POST /api/auth/login`: `{ "username", "password" }`
- `GET|POST /api/users`: super admin lists accounts or creates one `{ "username", "password", "role" }`
- `GET|POST /api/tournaments`
- `GET|POST /api/teams`
- `GET|POST /api/matches`
- `PATCH /api/matches/{id}/official`: `{ "official_id": "..." }` (admin assignment)
- `PATCH /api/matches/{id}/live`: `{ "home": 1, "away": 0 }`
- `POST /api/matches/{id}/finalize`: finalises the last saved live score (ignores the request body)
- `POST /api/tournaments/{id}/fixtures`
- `GET /api/standings?tournament_id={id}`
- `GET /healthz`

The monolith still uses its own PBKDF2 password hashing and hand-rolled JWT code; the microservices use bcrypt and `golang-jwt/v5`. Finalisation writes the result, both standings rows and an outbox event in one transaction, and a PostgreSQL trigger blocks later edits.

To run the monolith without Docker, set `DATABASE_URL`, `JWT_SECRET` and `NATS_URL`, then run `go run ./cmd/api` from `services/api-gateway`.

## Repository layout

```text
proto/              Protobuf contracts (Buf)
packages/           contracts (generated), servicekit, eventkit
services/           20 microservices, api-gateway (monolith), _template, shared builder/runtime images
deploy/             docker-compose.yml, nginx.conf, .env.example, build-services.ps1
tools/e2e/          end-to-end smoke test
docs/               project management and test planning documents
index.html, app.js, api.js, styles.css   frontend
```
