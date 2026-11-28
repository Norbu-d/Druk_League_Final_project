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

## Change log, Nov 2026

| ID | Date | Change | Reason | Impact | Decision / approver | Verification |
|---|---|---|---|---|---|---|
| CR-003 | By 10 Nov | Payment service removed from scope | Real payment processing is out of scope (proposal §2A) | 20 services remain without payment; Sprint 6 goal updated in Jira | Team | Jira sprint goal records the removal |
| CR-004 | 27 Nov | Namgay's frontend work handed to Norbu; the frontend stays plain HTML/CSS/JS (no React before submission) | Capacity before the 3 Dec deadline | RACI updated; landing page and workspace rebuilt by Norbu | Team (Norbu, Namgay) | `progress.md` §54–55 |
| CR-005 | 28 Nov | The BFF exposes Registration, Player and Venue, public read-only routes and viewer sign-up | Users had no usable journeys, and fixtures excluded every team created on the website (D-008) | New BFF routes and UI screens; security review: public routes use a service token, sign-up fixed to Viewer, rate-limited | Requested by Norbu as project owner | BFF tests, 36 real-stack checks (§59) |
| CR-006 | 28 Nov | Serverless functions delivered as OpenFaaS "dockerfile" functions on the of-watchdog, run under Compose locally, with `stack.yml` for OpenFaaS/faasd | faasd needs Linux and containerd and cannot run on the team's Windows machines | `functions/` (match-report, standings-csv, connector), CI coverage and image scanning | Norbu (owner of serverless in the proposal) | Unit tests 90–100%; 8 live checks |
| CR-007 | 28 Nov | Jira re-baselined: final sprint ends 3 Dec; past sprints closed; due dates inside their sprint; assignees follow the work | The board still ran to 11 Jan 2027 and did not match delivery | S2, S3, S4 closed; S5 started; S8 ends 3 Dec; 94 due dates, 8 assignees, 2 statuses updated | Approved by Norbu on 28 Nov | Jira board state on 28 Nov |
| CR-008 | 28 Nov | Every service image rebuilt from source before the demo | Deployed images dated from 10 Nov (D-010, D-011) | All containers recreated from the current commit | Team | Pending until the rebuild completes |
