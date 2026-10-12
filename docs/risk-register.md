# SWE303 — Risk Register

| ID | Risk | Likelihood / impact | Mitigation | Owner / trigger |
|---|---|---|---|---|
| R1 | Semester time is insufficient for 20 services, integrations, and the UI | High / High | Deliver the documented vertical-slice MVP first; keep service contracts versioned; treat service split as next milestone | Team; review each weekly checkpoint |
| R2 | Final scores are disputed or entered against the wrong match | Medium / High | Restrict score writes to assigned officials; use one-time finalization, audit event, and immutable database guard | Backend; any score dispute |
| R3 | Admin account or JWT secret is left at demo defaults | Medium / High | Require deployment secrets in production; replace `.env.example` values before deployment; do not commit `.env` | Deployment owner; before demo/public hosting |
| R4 | Two matches are scheduled for a team or venue at the same time | Medium / High | Reject same-kickoff team/venue clashes; add duration and venue-availability rules before real tournament use | Tournament owner; fixture planning |
| R5 | NATS or downstream consumers are unavailable after result finalization | Medium / Medium | Commit a transactional outbox with the result; retry JetStream publication and use stable message IDs | Backend; NATS outage/reconnect |
| R6 | Shared MVP database creates coupling between future services | High / Medium | Keep domain protobuf contracts independent; move to database-per-service when extracting services | Architecture owner; service-split milestone |
| R7 | UI shows illustrative values that are mistaken for live records | Medium / High | Label demo metrics; replace mock data screen by screen; disclose limitations in demo | Frontend owner; before external demo |
| R8 | Docker environment is unavailable on the development host | Medium / Medium | Keep API buildable with Go toolchain; document Compose requirements; validate deployment on a host with Docker Engine | Deployment owner; before release |
| R9 | Automated quality gates are absent | High / High | Maintain the test plan and traceability matrix; implement unit/integration/e2e coverage and CI as a release gate | QA owner; before final submission |

## Scope decision

Prometheus/Grafana dashboards, full security-scanning CI, and real AWS Lambda deployment remain stretch items. Basic OpenTelemetry tracing, automated core tests, and local NATS consumers are core follow-up work because they support the project's resilience and event-driven learning outcomes.
