# DrukLeague Project Charter

**Document:** SWE303 project charter
**Version:** 1.0
**Prepared:** 5 November 2026
**Status:** Draft for team and course review

## 1. Project purpose

DrukLeague is a web-based tournament operations platform for Bhutan's inter-dzongkhag sports competitions. It is intended to replace scattered phone calls, paper score sheets, and messaging-app updates with a shared workflow for tournament setup, team registration, fixtures, match scoring, final results, and standings.

The project is a semester MVP and learning deliverable for microservices/serverless development, software testing and quality assurance, and software project management. The project targets a 20-service architecture, while the implementation is staged: the current product includes a working Go API and browser UI alongside service modules, and gateway/BFF and service integration remain incomplete. This distinction is part of the agreed scope baseline.

## 2. Objectives and success measures

The team will:

1. Deliver a reproducible local MVP for tournament operations using Go, a browser-based interface, PostgreSQL, Docker Compose, and NATS where required by the implemented event flows.
2. Demonstrate core workflows for tournament and team records, fixture generation, official assignment, live scoring, result finalisation, and standings.
3. Preserve the rule that only the assigned match official finalises a result and that finalised results cannot be silently edited.
4. Build toward the documented 20-service target using versioned contracts and staged domain ownership; report unimplemented gateway, BFF, integration, and service capabilities accurately.
5. Produce reviewable SWE303 and SWE302 evidence: scope, stakeholders, architecture decisions, risks, sprint records, traceability, test results, defects, and a demo/run guide.

Success is assessed through demonstrable workflows, reproducible run instructions, automated tests and CI evidence where implemented, traceability from requirements to tests, and course/team review. No numeric coverage or full-service completion is claimed by this charter; those are tracked as quality gates and remaining work in the test plan and progress log.

## 3. Scope baseline

### In scope

- Web MVP for tournament operations and public fixture/result/standings viewing.
- User roles and access controls for tournament administrators, match officials, team managers, and viewers, subject to the current implementation and documented limitations.
- Team and tournament records, fixture generation, official assignment, live score updates, finalisation, and standings.
- Go service contracts and staged service implementation, PostgreSQL persistence, Docker Compose, and NATS/JetStream event flows where specified by the architecture.
- Unit, integration, contract, and end-to-end quality work, plus SWE303 planning and submission evidence.

### Out of scope for the semester MVP

- Paid production hosting or cloud deployment, production scaling, and native mobile applications.
- Real SMS delivery, payment processing, and production-grade multi-region operations.
- Treating all 20 services, gateway/BFF, or serverless functions as complete before their implementation and verification are recorded.
- Prometheus/Grafana dashboards and other stretch items unless the team formally reprioritises them.

## 4. Key deliverables

- Working local MVP and documented startup/demo instructions.
- Twenty service modules/contracts as the architecture target, with implementation and verification status tracked per service.
- Core tournament, fixture, score, finalisation, and standings workflows.
- SWE302 test plan, requirement-to-test traceability, automated test/CI evidence, and defect records.
- SWE303 project documentation: this charter, stakeholder analysis, WBS, architecture decisions, risk register, change control, sprint evidence, final report, and demo materials.

## 5. Stakeholders and responsibilities

| Stakeholder | Responsibility / interest |
|---|---|
| Bhutan Olympic Committee / national organizer | Tournament oversight, competition rules, and result-correction policy input |
| Dzongkhag sports administrators | Local team registration and venue coordination workflows |
| Match officials | Assigned-match scoring and result finalisation |
| Team managers | Team information, fixtures, and standings |
| Players and supporters | Public fixture and result information |
| Course lecturers/evaluators | Review learning outcomes, scope, architecture, testing evidence, and project documents |
| Student development team | Build, test, document, review risks, and demonstrate the semester MVP |

The named student team is Tandin Wangchuk, Norbu Dendup, Sonam Choki, and Namgay Lhamo. Detailed service ownership and task assignments are managed in Jira and the sprint records; this charter does not replace those assignments.

## 6. Approach and governance

The team works in eight planned two-week Scrum sprints. Jira is the source of record for sprint assignments and workflow. The WBS and `progress.md` record project-level scope and verified implementation progress. Weekly checkpoints cover completed work, next work, risks, and decisions. Architecture or scope changes are recorded in `change-control.md`, including impact and decision owner. Defects and their evidence are recorded in `defect-log.md`.

The architecture is staged: domain contracts and service boundaries support the 20-service target, while working behavior and test evidence are reported at the level actually implemented. The accepted architecture decision and open design choices are documented in `architecture-decision-mvp.md` and `progress.md`.

## 7. Milestones

The project uses milestone labels rather than calendar dates in this charter because the Jira sprint dates are being corrected by the project administrator. Jira remains the schedule source of record once corrected.

| Milestone | Exit evidence |
|---|---|
| M1 — Project baseline | Scope, stakeholders, WBS, and architecture target reviewed |
| M2 — Runnable MVP | API, PostgreSQL, NATS as applicable, and browser UI run together from documented local setup |
| M3 — Service architecture | Planned service APIs, data ownership, gateway/BFF routing, and integrations implemented or explicitly recorded as remaining |
| M4 — Quality gate | Planned tests executed, defects triaged, and CI evidence reviewed |
| M5 — Submission and demo | Project report, reproducible demo, and known limitations reviewed |

## 8. Assumptions and constraints

- The semester project is delivered by a four-person student team and assessed against WEB303, SWE302, and SWE303 learning outcomes.
- The project budget is Nu. 0 and the target development/deployment environment is local, free/open-source tooling with Docker Compose.
- The proposal's 20-service target remains the scope baseline; delivery is staged and risks are managed through the WBS and sprint planning.
- The current domain is football-style tournament scoring; broader sports formats and production rules require further decisions.
- Local Docker availability and machine disk/RAM capacity can constrain full-system verification.
- Stakeholder approvals, course acceptance, and any policy decisions are not presumed by this document.

## 9. Major risks and responses

- **Time and scope:** the 20-service ambition may exceed semester capacity. Prioritise a demonstrable vertical slice, track unfinished scope, and review progress each week.
- **Result integrity:** disputed or incorrect scores can undermine trust. Restrict scoring to assigned officials, make finalisation explicit, retain audit evidence, and resolve the correction policy before real use.
- **Security configuration:** development secrets must not be used in deployment. Require unique deployment secrets and document the setup.
- **Event delivery:** downstream services may be unavailable after finalisation. Use the outbox and bounded retry mechanisms where implemented; verify outage/recovery behavior as follow-up work.
- **Verification environment:** unavailable Docker or constrained host resources can block full integration checks. Run module tests/CI and document which container checks remain outstanding.
- **Scope and architecture uncertainty:** service ownership, registration rules, identity mapping, and other decisions can cause rework. Record decisions and impacts in the architecture and change-control documents.

The detailed risk owners, triggers, and mitigations are maintained in `risk-register.md`.

## 10. Acceptance and approval

Deliverables are ready for team/course review when their stated behavior is implemented or clearly labelled as incomplete, relevant test evidence is recorded, known limitations and defects are visible, and the demo can be followed from the documented setup. Jira issue acceptance criteria remain the detailed acceptance source for individual assignments.

This charter is a project baseline draft, not evidence of sponsor or lecturer approval. Reviewers and approval dates should be added only after those reviews occur.

| Review role | Reviewer | Decision/date |
|---|---|---|
| Student project team | Tandin Wangchuk, Norbu Dendup, Sonam Choki, Namgay Lhamo | Pending review |
| SWE303 lecturer/evaluator | To be confirmed | Pending review |
| Project sponsor / competition representative | To be confirmed | Pending / not yet obtained |
