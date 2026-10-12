# SWE302 — Test Plan

## Objective and scope

Define the quality checks for the DrukLeague MVP. The API currently builds, but the Docker runtime and all runtime behavior still require execution on a host with Docker Engine and PostgreSQL/NATS available. This plan records expected checks; it is not evidence that they have been executed.

## Test levels

- **Unit:** PBKDF2 password verification, JWT claims/expiry, standings calculations, fixture round-robin generation, request validation.
- **Integration:** PostgreSQL migrations and transactions; role-scope queries; NATS JetStream outbox publication and reconnect.
- **End-to-end:** Admin creates tournament/teams/users, assigns official, official records live score and finalizes, viewer reads standings.
- **Resilience:** NATS outage/reconnect; database unavailable at startup; service timeout/circuit breaker after service split.
- **Accessibility/usability:** readable text, keyboard modal control, validation feedback, responsive layout and color contrast.

## High-priority scenarios

| ID | Scenario | Setup / action | Expected result | Priority |
|---|---|---|---|---|
| T-01 | Login accepts valid credentials | Bootstrap admin; POST correct username/password | 200 with signed token and role; password never returned | P1 |
| T-02 | Login rejects invalid credentials | POST wrong password | 401 with generic message; no token | P1 |
| T-03 | Role-based write isolation | Match official and viewer attempt to create a team or tournament; tournament admin creates both with a dzongkhag | Non-admin writes return 403; admin writes persist the selected dzongkhag | P1 |
| T-04 | Official-only live score | Viewer or unassigned official patches a match | 403/409; score unchanged | P1 |
| T-05 | Assigned official updates live score | Assign official; PATCH non-negative score | 200; match is live; standings unchanged | P1 |
| T-06 | Finalized result is immutable | Assigned official finalizes; attempt second finalization and direct SQL result mutation | API returns conflict; database rejects mutation | P1 |
| T-07 | Finalization updates standings atomically | Finalize win/draw/loss in separate matches | Played, W/D/L, goals, and points update exactly once | P1 |
| T-08 | Team kickoff clash is blocked | Schedule a team twice at same kickoff | 409; no duplicate fixture | P1 |
| T-09 | Venue kickoff clash is blocked | Schedule two games at same kickoff and venue | 409; no second fixture | P1 |
| T-10 | Round-robin fixtures | Register an odd number of teams; generate fixtures | Each pair plays once, no self-match; one bye each round | P1 |
| T-11 | Duplicate fixture generation is blocked | Generate twice for same tournament | First succeeds; second returns conflict | P1 |
| T-12 | Outbox survives NATS outage | Stop NATS; finalize match; restore NATS | DB result commits; pending event publishes after recovery | P1 |
| T-13 | Viewer reads public data | Viewer requests tournaments, teams, matches, standings | 200 read response; create/update calls denied | P2 |
| T-14 | Oversized or malformed input | Send malformed JSON or body over 1 MiB | 400; server remains available | P2 |
| T-15 | UI form feedback | Submit invalid and valid form; keyboard-only modal flow | Clear feedback, focus remains understandable, modal closes on success | P2 |

## Entry and exit criteria

- Entry: clean Compose environment, migrations applied, secrets supplied, bootstrap account available.
- Exit: all P1 scenarios pass, no open critical/high defects, database result immutability verified, deployment instructions reproduce the demo.
- Record actual run date, environment, build commit, result, defect IDs, and evidence for each executed scenario before claiming a pass.

## Current execution status

No automated or manual scenarios are marked passed in this repository. The API has been compiled and JavaScript syntax and Compose configuration have been checked; Docker image/runtime integration could not be checked because Docker Engine was not running on the implementation host.
