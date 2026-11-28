# SWE302 — Requirements Traceability Matrix

Status on 28 Nov 2026. "Verified" means an execution is recorded in [test-execution-report.md](test-execution-report.md) or `progress.md` with its date.

| ID | Requirement | Design / implementation | Verification | Status |
|---|---|---|---|---|
| R-01 | Users authenticate before using protected APIs | `services/auth`, `services/user`, BFF session cookies and refresh | T-01, T-02; BFF tests; acceptance §59 | Verified |
| R-02 | Four roles with distinct permissions (Super Admin, Match Official, Team Manager, Viewer) | Role checks in each service's `internal/app`; `role-permissions.js` for the UI | T-03, T-04, T-13, T-17; 36 acceptance checks | Verified |
| R-03 | Teams, tournaments and venues carry a dzongkhag; no dzongkhag-scoped admin role | Dzongkhag service; Team, Tournament, Venue fields | Unit tests; acceptance (venue in Trongsa) | Verified |
| R-04 | The admin creates tournaments; teams are entered and approved | Tournament, Team, Registration services; BFF `/api/registrations` | T-15 (UI), acceptance §59 (approve, reject with reason, no re-entry) | Verified |
| R-05 | Fixture generation uses approved teams only, prevents duplicates and books venues in the dzongkhag | `services/fixture` | T-10, T-11; live: 1 fixture, approved teams only, QA venue | Verified (T-08/T-09 by unit tests) |
| R-06 | Live scores do not change standings until finalised | Score `UpdateLiveScore`; Standings consumer | T-05 | Unit tests; live pre-finalisation check not recorded |
| R-07 | Only the assigned official finalises, once | Match assignment; Score caller check; immutable score | T-04, T-06; live: second finalise → 409 | Verified |
| R-08 | Final results update standings exactly once | Score outbox; Standings idempotent consumer | T-07; live standings 3/0 | Verified |
| R-09 | `MatchFinalized` delivered durably and processed idempotently | eventkit outbox, durable consumers, DLQ | T-07, T-12 | Delivery verified live; outage/replay evidence open (D-005) |
| R-10 | Teams, fixtures, results and standings readable through the web app | BFF lists and public routes; `api.js` | T-13; mocked UI for every role; real guest view | Verified |
| R-11 | Twenty gRPC services with versioned contracts | `proto/drukleague`, `services/*` | Buf lint/build/breaking in CI; coverage ≥ 80% for all 20 | Verified |
| R-12 | Resilience: deadlines, retries, circuit breaker, graceful degradation | `packages/servicekit`, BFF list degradation, function degradation | Unit tests (servicekit 83.5%); function degradation tests | Unit-verified; Docker fault injection open (D-005) |
| R-13 | Reproducible local deployment | `deploy/docker-compose.yml`, Dockerfiles, nginx | Compose config check; stack run on 28 Nov | Partly verified; full rebuild in progress (D-011) |
| R-14 | Quality and project-management evidence accompany submission | `docs/` | Review | Updated 28 Nov (plan, quality plan, test report, lessons learned) |
| R-15 | Serverless, event-driven functions | `functions/` (match-report, standings-csv, connector, `stack.yml`) | Unit tests 90–100%; 8 live checks | Verified |
| R-16 | Visitors read public data; anyone can create a Viewer account | BFF `/api/public/*`, `/api/auth/signup`; nginx rate limits | BFF tests (service token, ids hidden, role fixed); acceptance §59 | Verified |
| R-17 | Squads: team managers maintain their own players; others read only | Player service via the BFF; Squads page | Acceptance §59 (403 for other managers and viewers, 400 for shirt > 99, 409 for a duplicate shirt) | Verified after D-010 |
