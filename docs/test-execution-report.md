# SWE302 — Test Execution Report

Covers executions from 27–28 Nov 2026 on branch `feature/norbu-landing-page` (uncommitted working tree based on `667f966`). Every result below was produced by a command or script in this repository; nothing is copied from the plan. Test cases are defined in [swe302-test-plan.md](swe302-test-plan.md); defects are in [defect-log.md](defect-log.md).

## 1. Summary

| Level | Tool | Scope | Result |
|---|---|---|---|
| Unit + integration (Go) | `go test`, `scripts/coverage.sh 80` with PostgreSQL 17 (`TEST_DATABASE_URL`), Linux | 25 modules (packages, services, BFF) | **All pass; every module ≥ 80%** (§2) |
| Serverless functions | `go test -cover` | match-report, standings-csv, connector | **Pass:** 91.6%, 90.5%, 100% |
| BFF HTTP API | `go test` (fake gRPC clients) | Every route, including the new public, sign-up, registration, player and venue routes | **Pass** |
| Frontend policy | `node --test tests/frontend/role-permissions.test.js` | Role pages, actions, API allowlist | **10/10 pass** |
| UI rendering per role | Headless Chrome with a mocked `/api` against the real nginx container | Guest, sign-up, admin, official, manager, viewer at 1440 px and 390 px | **No script errors**; 27 screenshots reviewed, 4 layout defects fixed |
| Real-stack acceptance | Node scripts against the real BFF on the test-only database | 36 role and permission checks | **36/36 after D-010** (§4) |
| Event-driven + serverless, live | Node script and connector logs | Finalise → JetStream → Standings → connector → function | **8/8 pass** (§5) |
| Browser E2E | Playwright 1.49 (`tests/e2e`) | Guest, sign-up, admin journeys | **Not yet executed:** the first run hit a Docker Desktop engine outage (the app was unreachable). Rerun pending (§6). |
| Performance | k6 (`tests/performance`) in CI | Login and public standings | Runs in CI on each PR; no local run on 28 Nov |
| Security scanning | gosec, govulncheck, Trivy in CI | Go code, dependencies, all images | Run in CI; the BFF and function images were added to the Trivy job on 28 Nov (D-013) |
| Web security scan | OWASP ZAP baseline | Public site | **Pending** (needs the stack running) |

## 2. Coverage by module (28 Nov 2026, Linux container, PostgreSQL 17, the same script as CI)

| Module | Coverage | Module | Coverage |
|---|---:|---|---:|
| packages/eventkit | 90.1% | services/match | 92.9% |
| packages/servicekit | 83.5% | services/media | 80.4% |
| services/analytics | 94.1% | services/notification | 93.1% |
| services/api-gateway | 100.0% | services/official | 96.8% |
| services/audit-log | 97.6% | services/player | 97.5% |
| services/auth | 94.0% | services/registration | 95.0% |
| services/bff | 86.8% | services/report | 95.4% |
| services/configuration | 91.7% | services/score | 90.9% |
| services/dzongkhag | 97.1% | services/search | 91.8% |
| services/fixture | 85.2% | services/standings | 93.1% |
| services/team | 97.0% | services/tournament | 96.6% |
| services/user | 94.6% | services/venue | 95.8% |

`packages/contracts` holds generated code and `tools/event-admin` is an operator tool; neither has tests. **Objective Q1 (≥ 80% per service) is met.** On Windows the same script reports false "TEST-FAIL" results because the OS locks the test binary (`unlinkat … being used by another process`) and the PostgreSQL tests are skipped; the Linux run is the reference.

## 3. Test-plan cases executed

| Case | Description | How | Result |
|---|---|---|---|
| T-01 / T-02 | Login valid / invalid | Acceptance script; BFF tests | Pass |
| T-03 | Role-based write isolation | Acceptance: manager/viewer refused on venues, fixtures, approvals, other teams' squads | Pass (after D-010) |
| T-04 | Only the assigned official scores | Live script: the assigned official saves and finalises | Pass |
| T-06 | Final result is immutable | Second finalise returns `409 Score is already finalised` | Pass |
| T-07 | Finalisation updates standings once | Live: standings show 3 pts for the winner, 0 for the loser | Pass |
| T-10 | Round-robin fixtures | Generated 1 fixture for 2 approved teams, at the dzongkhag's venue | Pass |
| T-13 | Viewer reads public data; writes denied | Public routes return 200 without sign-in; viewer write → 403 | Pass |
| T-14 | Malformed input | Sign-up with a `role` field → 400; shirt 120 → 400; reject without a reason → 400 | Pass |
| T-16 | Cross-service validation | A rejected team cannot re-enter (409); fixtures exclude rejected and pending teams | Pass |
| T-17 | Role-based UI visibility | Frontend tests plus the mocked UI run for every role | Pass |
| New | Serverless event path | `MatchFinalized` → connector → match-report → `drukleague.functions.match-report` | Pass |
| New | Serverless HTTP path | `GET /fn/standings-csv` → ranked CSV; a bad id → 400 | Pass |

Cases not executed on 28 Nov: T-05 (pre-finalisation standings check), T-08/T-09 (clash checks; covered by unit tests only), T-11, T-12 (NATS outage, see D-005), T-15, T-18 to T-22 (covered by unit and integration tests in §2, with no separate acceptance record).

## 4. Real-stack acceptance (28 Nov 2026)

Script: `progress.md` §59. Data on the confirmed test-only database is labelled "QA Check 11271912". The first run gave 32/36: 4 squad-permission failures caused by the stale player image (D-010). After rebuilding that image, all 36 checks pass, and the duplicate-shirt rule (409) was confirmed separately.

## 5. Live event-driven and serverless checks (28 Nov 2026)

| Check | Result |
|---|---|
| The generated QA fixture is listed | Pass |
| Admin creates and assigns a match official | Pass |
| Official saves a live score, then finalises 2–1 | Pass |
| Finalised result cannot be changed (409) | Pass |
| Standings update asynchronously from the event | Pass (QA Manager FC 3, QA Admin XI 0) |
| `standings-csv` returns the updated table | Pass (`1,QA Manager FC 11271912,Trongsa,1,1,0,0,2,1,1,3`) |
| Connector invoked `match-report` for this match | Pass (log line with headline "QA Manager FC 11271912 2–1 QA Admin XI 11271912") |

Observation: in that announcement the "top of the table" part was empty, because the function ran before the Standings projection had applied the same event. This is expected eventual consistency. A short retry inside the function would remove it (tracked as an improvement, not a defect).

## 6. Pending executions

1. **Playwright E2E** (`cd tests/e2e && npx playwright test`, with `DRUK_ADMIN_PASSWORD` set for the admin specs): rerun once Docker Desktop is restarted and every image is rebuilt (D-011).
2. **OWASP ZAP baseline**: `docker run --rm -t ghcr.io/zaproxy/zaproxy:stable zap-baseline.py -t http://host.docker.internal:4173`, with the results recorded here.
3. **Fault recovery (D-005)**: stop NATS, finalise a match, restart NATS, confirm the outbox drains; replay one DLQ message with `tools/event-admin`.
