# SWE302 — Test Plan

## Objective and scope

Define quality checks for the DrukLeague MVP and extracted services. This plan records both expected checks and the limited manual gRPC/JetStream verification completed on 14 Oct 2026; it is not evidence that the full test suite has run.

## Test levels

- **Unit:** PBKDF2 password verification, JWT claims/expiry, standings calculations, fixture round-robin generation, request validation.
- **Integration:** PostgreSQL migrations and transactions; role checks; Score outbox publication; Match and Standings JetStream consumers; duplicate-event handling.
- **End-to-end:** Admin creates tournament/teams/users, assigns official, official records and finalizes a score, then a viewer reads eventually consistent standings.
- **Resilience:** NATS outage/reconnect; database unavailable at startup; service timeout/circuit breaker after service split.
- **Accessibility/usability:** readable text, keyboard modal control, validation feedback, responsive layout and color contrast.

## High-priority scenarios

| ID | Scenario | Setup / action | Expected result | Priority |
|---|---|---|---|---|
| T-01 | Login accepts valid credentials | Bootstrap admin; POST correct username/password | 200 with signed token and role; password never returned | P1 |
| T-02 | Login rejects invalid credentials | POST wrong password | 401 with generic message; no token | P1 |
| T-03 | Role-based write isolation | Match official and viewer attempt to create a team or tournament; tournament admin creates both with a dzongkhag | Non-admin writes return 403; admin writes persist the selected dzongkhag | P1 |
| T-04 | Official-only live score | Viewer or unassigned official patches a match | 403/409; score unchanged | P1 |
| T-05 | Assigned official updates live score | Assign official; send non-negative score to Score; read with `GetScore` | Score persists; standings remain unchanged before finalization | P1 |
| T-06 | Finalized result is immutable | Assigned official finalizes; attempt a second finalization, score update, and direct SQL mutation | API returns conflict; PostgreSQL trigger rejects direct update/delete | P1 |
| T-07 | Finalization updates standings eventually and once | Finalize win/draw/loss in separate matches; wait for consumer; republish one event with the same `event_id` | Score and outbox commit atomically; Standings reflects results asynchronously; duplicate event does not change totals | P1 |
| T-08 | Team kickoff clash is blocked | Schedule a team twice at same kickoff | 409; no duplicate fixture | P1 |
| T-09 | Venue kickoff clash is blocked | Schedule two games at same kickoff and venue | 409; no second fixture | P1 |
| T-10 | Round-robin fixtures | Register an odd number of teams; generate fixtures | Each pair plays once, no self-match; one bye each round | P1 |
| T-11 | Duplicate fixture generation is blocked | Generate twice for same tournament | First succeeds; second returns conflict | P1 |
| T-12 | Outbox survives NATS outage | Stop NATS; finalize match; restore NATS | DB result commits; pending event publishes after recovery | P1 |
| T-13 | Viewer reads public data | Viewer requests tournaments, teams, matches, standings | 200 read response; create/update calls denied | P2 |
| T-14 | Oversized or malformed input | Send malformed JSON or body over 1 MiB | 400; server remains available | P2 |
| T-15 | UI form feedback | Submit invalid and valid form; keyboard-only modal flow | Clear feedback, focus remains understandable, modal closes on success | P2 |
| T-16 | Cross-service match validation | Admin schedules teams from different tournaments and assigns a non-official user | Mismatch/non-official is rejected; both teams must belong to the selected tournament; only a `match_official` user can be assigned | P1 |
| T-17 | Role-based UI visibility | Evaluate the shared role permission policy for Super Admin, Match Official, Team Manager, Viewer, and an unknown role | Page access and action permissions match the four-role matrix; unknown roles fall back to Viewer | P2 |
| T-18 | Media upload/retrieval authorization and validation | With a migrated test database, upload an allowed image as an authenticated team manager, list it by match, retrieve the exact bytes, reject a foreign-owner delete, then delete as the uploader; reject missing auth and unsupported content | Metadata and binary agree, unauthorized deletion is denied, and unsupported content never creates a row/blob | P1 |
| T-19 | Analytics dashboard filters | Record representative finalized-match facts for two seasons/dzongkhags and query each filter independently | Tournament, distinct participating-team and finalized-match counts reflect only matching facts | P2 |
| T-20 | Configuration read/write authorization and validation | With a migrated PostgreSQL test database, read a missing and existing key; attempt writes anonymously, as Viewer, and as Super Admin; update an existing value and submit invalid keys/oversized values | Reads return the expected value or NotFound; writes require Super Admin; valid values persist and invalid input returns InvalidArgument | P1 |
| T-21 | Search validation, filters, and pagination | Search seeded team/player/tournament documents with mixed-case text, duplicate entity types, page sizes and continuation tokens; submit invalid filters/tokens | Matching entities are filtered and ordered; page tokens advance correctly; invalid input is rejected; PostgreSQL content fields map to titles/subtitles | P2 |
| T-22 | API Gateway token and HTTP helper unit tests | Run `go test ./cmd/api` in `services/api-gateway`; cover signed/expired/tampered tokens, strict JSON decoding, role predicate, nullable database values, safe error responses, and CORS preflight | Valid token round-trips; expired/altered tokens and unknown JSON fields are rejected; helpers return expected status, headers, and safe bodies | P2 |

Score and Standings use separate databases. No transaction spans them: Score commits its final result and outbox record in one transaction; the Standings consumer applies both team rows and records the `event_id` in one local transaction. Therefore T-07 checks eventual consistency and idempotency, not cross-database atomicity.

## Entry and exit criteria

- Entry: clean Compose environment, migrations applied, secrets supplied, bootstrap account available.
- Exit: all P1 scenarios pass, no open critical/high defects, database result immutability verified, deployment instructions reproduce the demo.
- Record actual run date, environment, build commit, result, defect IDs, and evidence for each executed scenario before claiming a pass.

## Current execution status

Manual checks completed on 14 Oct 2026: Match rejected a cross-tournament team (`InvalidArgument`); an assigned official updated a 2–1 score, finalized it, and read it through `GetScore`; an admin update was denied and a post-finalization update was rejected; direct SQL mutation was rejected by the score immutability trigger; the Match consumer marked the match completed; Standings loaded team metadata and applied the event; republishing the same `event_id` left each team's `played` total at 1. Match, Score and Standings built and ran in Compose. On 5 Nov, four Node unit tests for T-17 passed locally, and the role-policy test was added to CI. These tests exercise the shared permission policy, not browser-rendered E2E behavior. Outage/reconnect, dead-letter, win/draw/loss permutations, full UI E2E and performance scenarios remain pending.

On 9 Nov, Media upload/list/download/delete integration cases and Analytics filter tests ran against an isolated PostgreSQL 17 test database and passed. The full Go module suite also passed locally on Go 1.26.5. Buf 1.73.0 lint, build and breaking checks against the fetched `development` branch passed. Follow-up work added coverage cases; on commit `fdd5ce7`, GitHub Actions reported all eight checks successful, including the module coverage gate, Buf, security scans, containers and k6. T-18/T-19 had database-backed coverage in the earlier run; the authenticated browser workflow, full Compose run on `fdd5ce7`, Docker outage/reconnect, non-empty DLQ replay, win/draw/loss permutations and full browser E2E remain unverified. These checks are evidence for the SWE302 test plan; map them to the actual Jira acceptance criteria before changing issue status.

On 18 Nov, added Configuration authorization/persistence tests and Search application, gRPC mapping, and PostgreSQL filtering/pagination tests (T-20/T-21). `go test ./...` passed in both modules locally; PostgreSQL-backed cases were skipped because `TEST_DATABASE_URL` is not configured. GitHub CI must confirm those database tests and the 80% module coverage gate. On 27 Nov, API Gateway unit tests for T-22 and `go test ./...` passed; focused coverage was 5.9% for `cmd/api` and 100% for `internal/auditoutbox`, so the API command still needs broader tests. The parent SCRUM-24 was last recorded In Progress; refresh Jira and reconcile its full acceptance evidence before changing its status. Historical progress notes incorrectly referenced SCRUM-55; the user confirmed that issue does not exist, so it is not a valid dependency or status source.

On 27–28 Nov, executed results (coverage per module on Linux with PostgreSQL, BFF and frontend tests, 36 real-stack acceptance checks, 8 live event-driven and serverless checks) were recorded in [test-execution-report.md](test-execution-report.md). Browser end-to-end tests now live in `tests/e2e` (Playwright); their first execution is pending a Docker Desktop restart.
