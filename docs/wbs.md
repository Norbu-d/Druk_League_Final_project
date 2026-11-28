# SWE303 — Work Breakdown Structure

Status on 28 Nov 2026. Each work package maps to a Jira epic or story on the `SCRUM` board; "Evidence" points to where completion can be checked.

| WBS | Work package | Deliverables | Owner | Status | Evidence |
|---|---|---|---|---|---|
| 1 | Project initiation | Proposal, charter, stakeholder analysis, 20-service scope | Namgay | Complete | Proposal (6 Oct), `docs/project-charter.md`, `docs/stakeholder-analysis.md` |
| 2 | Architecture and contracts | Service boundaries, protobuf contracts, database per service, architecture diagrams | Tandin / Norbu | Complete | `proto/`, `docs/architecture.md`, Buf checks in CI |
| 3 | Identity and access | Auth, User, roles, JWT and refresh cookies, viewer sign-up | Tandin | Complete | `services/auth`, `services/user`, BFF sign-up tests |
| 4 | Teams and tournaments | Team, Player, Tournament, Registration with an approval workflow | Tandin | Complete | Services ≥ 95% coverage; acceptance §59 |
| 5 | Competition | Match, Score, Fixture, Standings, Venue; clash checks; immutable results | Norbu | Complete | Live checks in the [test execution report](test-execution-report.md) §5 |
| 6 | Operations services | Official, Dzongkhag, Notification, Audit Log, Media | Sonam | Complete (media UI open, D-002) | Services ≥ 80% coverage |
| 7 | Supporting services | Report, Search, Analytics, Configuration | Namgay | Complete (no UI for Search/Analytics) | Services ≥ 91% coverage |
| 8 | Events and resilience | JetStream outbox, durable consumers, DLQ, deadlines, retries, breaker | Norbu | Complete; live fault-recovery evidence open (D-005) | `packages/eventkit` 90.1%, `servicekit` 83.5% |
| 9 | Access layer | API gateway, BFF (including public routes) | Norbu / Sonam | Complete | `services/bff` 86.8%, `services/api-gateway` 100% |
| 10 | Serverless | match-report and standings-csv functions, connector, `stack.yml` | Norbu | Complete | `functions/README.md`; 90–100% coverage; live checks |
| 11 | Web application | Landing page, role workspace, registrations, squads, venues | Norbu (from 27 Nov), Namgay | Complete | `progress.md` §54–58; mocked UI run for every role |
| 12 | Quality engineering | Test plan, CI gates, E2E, performance, security, defect log, traceability | Namgay | In progress (Playwright and ZAP runs pending) | [quality-plan.md](quality-plan.md), [test-execution-report.md](test-execution-report.md) |
| 13 | Deployment | Docker Compose stack, nginx hardening, monitoring | Norbu / Namgay | In progress (image rebuild, D-011) | `deploy/`, `monitoring/` |
| 14 | Project management | Plan, Jira, risks, changes, sprint records, lessons learned | Namgay | Complete for submission; final retrospective pending | [project-management-plan.md](project-management-plan.md) |
| 15 | Closure | Final report, presentation and demo (Weeks 14–15) | All | Not started | — |

## Milestones

| Milestone | Target | Status |
|---|---|---|
| M1 Project baseline | Sprint 1 | Done |
| M2 Runnable MVP (Compose) | Sprint 2 | Done |
| M3 All 20 services with database per service and gRPC | Sprint 6 | Done |
| M4 Integration, events, serverless | Sprint 7 | Done |
| M5 Quality gate (all CI gates green, acceptance recorded) | Sprint 8 | In progress |
| M6 Submission and presentation (3 Dec deadline; presentation Weeks 14–15) | Sprint 8 | Pending |
