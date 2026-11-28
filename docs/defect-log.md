# SWE302 — Defect Log

Defects and delivery gaps found by review, CI and the real-stack acceptance runs. Last updated 28 Nov 2026. "Open" means the workflow is not yet implemented or its acceptance is not yet verified; it does not imply a production incident. Severity follows the triage policy below.

| ID | Summary | Severity | Status | Found by / evidence | Resolution |
|---|---|---:|---|---|---|
| D-001 | Dashboard and several screens showed illustrative values after sign-in | Medium | **Closed 27 Nov** | Review of `app.js` | The workspace rebuild removed every hard-coded figure; all screens read API data (`progress.md` §55). |
| D-002 | Media upload/retrieval lacks browser acceptance evidence | Medium | In progress | Integration tests passed on 9 Nov; there is no UI for media yet | Add a media screen or a recorded API acceptance run. |
| D-003 | Full-stack acceptance on the current revision was missing | High | **Closed 28 Nov** | — | Real-stack acceptance: 36 checks (§59), plus 8 live event checks, all passing after D-010 was fixed. |
| D-004 | Some service contracts and cross-service ownership were incomplete | High | **Partly closed 28 Nov** | Review | Registration, Player and Venue are now reachable through the BFF and used by the UI; Search and Analytics remain without UI screens. |
| D-005 | Fault recovery and dead-letter replay lack live end-to-end evidence | High | Open | eventkit has embedded-JetStream tests; `tools/event-admin` exists | Run a controlled NATS stop/restart and a DLQ replay on the Compose stack and record it. |
| D-006 | `/auth/refresh` was missing from the frontend's API allowlist, so sessions never renewed | High | **Closed 27 Nov** | Code review during the UI rebuild | Allowlist fixed; regression test in `tests/frontend/role-permissions.test.js`. |
| D-007 | A scheduled match could never be scored: the score button appeared only on live matches | High | **Closed 27 Nov** | UI review against the Score service rules | The assigned official now sees Start scoring on scheduled matches. |
| D-008 | Teams created on the website were never registered, so fixture generation (approved registrations only) left them out | High | **Closed 28 Nov** | Backend review (`services/fixture` reads approved registrations; the BFF had no registration route) | The BFF registers each new team; admin entries are approved automatically; there is an approval queue in the UI. Verified live: fixtures used only approved teams. |
| D-009 | Visitors saw no data, and nobody could sign up | Medium | **Closed 28 Nov** | User feedback | Public read-only routes (service token, account ids removed) and viewer-only sign-up; BFF tests and real-stack checks. |
| D-010 | The running player container accepted players from a viewer and from another team's manager, and accepted shirt number 120 | **High (security)** | **Closed 28 Nov** | Real-stack acceptance run, `progress.md` §59 | The image had been built on 10 Nov from older code. Rebuilt from source; all squad permission checks pass. Three invalid players remain in the QA test team. |
| D-011 | Every other service image was built on 10 Nov, older than its source | High | In progress | Image build dates against `git log`, 28 Nov | Full rebuild started 28 Nov; the first parallel build lost the Docker Desktop engine connection. Rebuild with `COMPOSE_PARALLEL_LIMIT=4`, recreate the containers, then rerun acceptance. |
| D-012 | `.dockerignore` had two allowlist entries joined on one line, and the function connector was excluded from build contexts | Low | **Closed 28 Nov** | Connector image build failure | Split the line; added `functions/connector`. |
| D-013 | CI built every service image except the BFF, so the BFF was never Trivy-scanned | Medium | **Closed 28 Nov** | Review of `.github/workflows/go-ci.yml` | Added `bff` and the three function images to the build and scan job. |
| D-014 | Real-stack test data with invalid values exists ("Bad" #120 in QA Manager FC 11271912) | Low | Open | Side effect of D-010 | Remove by SQL on the test database, or reset the test database before the demo. |

## Triage policy

P1 / High: blocks a demo or risks unauthorised or corrupt results. P2 / Medium: affects a major workflow or has a workaround. P3 / Low: cosmetic or test data. A defect is closed only after its reproduction passes on a recorded revision. Anything that depends on an external or design decision stays open.

## Summary on 28 Nov 2026

14 defects logged: 9 closed, 1 partly closed, 4 open or in progress. Open High: D-005 (fault-recovery evidence) and D-011 (image rebuild).
