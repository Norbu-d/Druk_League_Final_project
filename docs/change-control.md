# SWE303 — Change Control

## Change request record

| Field | Entry |
|---|---|
| Change ID | CR-001 |
| Request | Use Go for the service architecture while retaining the DrukLeague UI and supplied logo. |
| Reason | The earlier 22-service NestJS/mobile draft was dropped. The current approved proposal specifies 20 Go microservices. |
| Impact | Go API MVP, PostgreSQL, NATS JetStream, versioned protobuf contracts, and a staged 20-service implementation. |
| Decision | Accepted for the MVP baseline. |
| Risks | The consolidated API is not the final target topology; do not present the MVP or current service shells as 20 completed business services. |
| Verification | API build succeeds; Compose syntax is valid. Docker runtime currently needs recovery after a stalled multi-image build. |

| Field | Entry |
|---|---|
| Change ID | CR-002 |
| Request | Continue toward all 20 independently deployable services. |
| Reason | Project owner confirmed the proposal scope on 7 Oct 2026. |
| Impact | Twenty service contracts/modules, service-owned database schemas, gRPC domain APIs, gateway/BFF routing, integration, tests, and deployment are required. |
| Decision | Accepted by the project owner; tutor approval is not recorded as required because this matches the current proposal. |
| Risks | Twenty services in one semester is high risk; the present modules are shells, and only Dzongkhag Get/List has a domain handler. |
| Verification | Buf lint/build and code generation pass; all 20 Go shells compile; full Docker build/runtime remains unverified. |

## Future change workflow

1. Give each requested change an ID and a short problem statement.
2. Identify affected requirements, code, data, tests, schedule, and risks.
3. Record who requested and who approved the change.
4. Implement only after the scope decision is recorded.
5. Update this log, WBS, risk register, traceability matrix, and user documentation.