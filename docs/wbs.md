# SWE303 — Work Breakdown Structure

| WBS | Work package | Deliverable | Status |
|---|---|---|---|
| 1 | Problem and scope baseline | Project overview; 20-service target confirmed | Complete |
| 2 | UX prototype | Responsive DrukLeague screens and supplied logo | Complete |
| 3 | Data model | MVP PostgreSQL migration plus 20 service-specific schemas | MVP and service shells complete; domain ownership integration pending |
| 4 | Identity and access | Login, signed sessions, roles, dzongkhag reference data | MVP complete; Auth/User service extraction pending |
| 5 | Tournament operations | Create/list tournaments, register/list teams | MVP complete; service extraction and team-user role pending |
| 6 | Fixture operations | Clash-aware match creation and round-robin generation | MVP complete; Fixture/Venue extraction pending |
| 7 | Score workflow | Live updates, official assignment, immutable finalization, standings transaction | MVP complete; Match/Score/Standings extraction pending |
| 8 | Event delivery | Transactional outbox and JetStream publication | Publisher complete; consumer functions pending |
| 9 | Service architecture | Twenty gRPC services, API gateway and BFF | Contracts and Go shells generated; Dzongkhag Get/List implemented; 19 domain APIs and gateway/BFF pending |
| 10 | Frontend integration | Login and core write/read paths | Partial; dashboards, officials, venues, and reports need API data |
| 11 | Quality engineering | SWE302 plan, traceability, automation, CI | Documentation underway; automated suite and CI pending |
| 12 | Submission and demo | Reproducible deployment, demo script, known limitations | Compose configured; engine stopped responding during full service image build |

## Milestones

1. **M1 — Project baseline:** 20-service scope, stakeholders, and architecture decision recorded.
2. **M2 — Runnable MVP:** API + PostgreSQL + NATS + browser UI can run together from Compose.
3. **M3 — Service architecture:** all 20 service domain APIs, databases, gateway/BFF routing, and integration work.
4. **M4 — Quality gate:** test plan executed, defects triaged, CI green.
5. **M5 — Submission:** reports, demo script, and known limitations reviewed.