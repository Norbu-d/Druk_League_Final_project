# Scrum plans, reviews and retrospectives

This record is based on the eight sprint goals and dates on the Jira board (`SCRUM board`, 9 Nov 2026). Dates below are copied as UTC from Jira. Sprint state is taken from the board and may not match the calendar dates; Jira remains the scheduling source of truth. This document does not invent team decisions or claim unrecorded reviews happened.

## Sprint plan

| Sprint | Planned dates (UTC) | Goal from Jira | Board state |
|---|---|---|---|
| 1 — Setup, Infra & CI/CD | 21 Sep–5 Oct 2026 | Project setup, repository, Compose, CI/CD, monitoring and protobuf definitions | Closed |
| 2 — User & Gateway | 5–19 Oct 2026 | User, Auth, API Gateway and BFF | Future |
| 3 — Team & Tournament | 19 Oct–2 Nov 2026 | Team, Registration and Tournament | Future |
| 4 — Venue & Fixture | 2–16 Nov 2026 | Venue, Official, Dzongkhag and Fixture | Future |
| 5 — Match & Score | 16–30 Nov 2026 | Match, Score and Standings | Future |
| 6 — Notify & Media | 30 Nov–14 Dec 2026 | Notification, Media and Report; payment removed from scope | Future |
| 7 — Search & Config | 14–28 Dec 2026 | Search, Analytics, Audit Log and Configuration | Future |
| 8 — Integration & QA | 28 Dec 2026–11 Jan 2027 | System, end-to-end and performance testing plus documentation | Future |

### Dependency reorder

The 22 Oct design review found that fixture clash checks depend on Venue. The Jira plan orders Venue & Fixture in Sprint 4 before Match & Score in Sprint 5. This puts the dependency before downstream match/score integration. This records the plan order; it does not claim that Jira sprint states or dates were changed during this update.

## Sprint review and retrospective evidence

| Sprint | Review evidence | Retrospective evidence |
|---|---|---|
| 1 | Jira marks the sprint closed; `progress.md` records initial repository, Compose, CI and contracts setup. No sprint review notes or stakeholder acceptance record are present in the repository. | No team retrospective notes are present. Add team observations and owners after the group confirms them. |
| 2–8 | Jira currently marks these sprints Future. Review outcomes are not yet available. | Retrospectives must be written after each sprint with the team's input. |

## Review template

For each sprint, record the date and attendees, planned versus accepted Jira work, demo and stakeholder feedback, defects carried forward, and any backlog or scope changes. Link to commits/PRs and acceptance evidence.

## Retrospective template

For each sprint, ask what helped, what slowed the team, and one measurable improvement for the next sprint. Record agreed actions with an owner and due sprint. Do not treat repository commit history as a substitute for team feedback.
