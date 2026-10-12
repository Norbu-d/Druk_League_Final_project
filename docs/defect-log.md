# SWE302 — Defect Log

| ID | Summary | Severity | Status | Reproduction / evidence | Planned resolution |
|---|---|---:|---|---|---|
| D-001 | Dashboard still shows illustrative aggregate metrics after API sign-in | Medium | Open | Sign in with an empty or populated API and compare overview values with PostgreSQL counts | Add dashboard aggregate endpoint and bind every metric/chart to returned values |
| D-002 | Officials and venue screens still use prototype data and have no API workflows | Medium | Open | Create official or venue through the database/API and view those screens | Add official listing/assignment UI and venue persistence endpoints |
| D-003 | Full Docker build and service runtime have not been verified | High | Blocked by Docker engine | Initial MVP stack started and API/web returned HTTP 200; a later 20-image Compose build stalled and Docker CLI calls stopped responding | Restart/recover Docker, run deploy/build-services.ps1, launch Compose and record health |
| D-004 | Twenty domain gRPC service APIs are not running yet | High | Open / planned gap | Twenty Go shells/contracts compile; only Dzongkhag Get/List has domain RPC handlers; other modules expose health | Implement service domain handlers and repositories, connect gateway/BFF clients, then run integration checks |
| D-005 | Notification and fixture-generation consumers are not implemented | Medium | Open / planned gap | NATS stream receives events but no consumer services subscribe | Add durable consumers and idempotent processing |

## Triage policy

P1 blocks a demo or risks unauthorized/corrupt results. P2 affects a major workflow or usability but has a workaround. P3 is cosmetic. Every resolved item should include a reproduction path and verification evidence.