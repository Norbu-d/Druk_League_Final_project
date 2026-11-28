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

## Board state after the 28 Nov 2026 re-baseline

| Sprint | Dates (UTC) | State on 28 Nov | Delivered (merged PRs / commits in the window) |
|---|---|---|---|
| 1 Setup, Infra & CI/CD | 21 Sep–5 Oct | Closed | 0 / 0 (the repository starts on 12 Oct) |
| 2 User & Gateway | 5–19 Oct | Closed 28 Nov | 0 / 3 |
| 3 Team & Tournament | 2–9 Nov | Closed 28 Nov | counted in Sprint 4's window |
| 4 Venue & Fixture | 2–16 Nov | Closed 28 Nov | 32 / 108 |
| 5 Match & Score | 16–30 Nov | Active | 17 / 31 (to 28 Nov) |
| 6 Notify & Media | 10 Nov–14 Dec | Closed (dates cannot be edited on a closed sprint) | — |
| 7 Search & Config | 9 Nov–22 Dec | Closed (dates cannot be edited on a closed sprint) | — |
| 8 Integration & QA | 9 Nov–3 Dec | Active (end date moved from 11 Jan 2027 to 3 Dec) | — |

All issues in Sprints 2–5 were Done when the sprints were closed, so nothing moved back to the backlog. Due dates now fall inside each issue's sprint; open work is due 3 Dec. See change CR-007.

## Sprint review evidence (from the repository)

| Sprint | Reviewed increment | Evidence |
|---|---|---|
| 4 | 20 service modules with database per service; PR workflow with required checks introduced | 32 merged PRs; `progress.md` §24–49 (6–13 Nov) |
| 5 | Web client role flows, BFF, signed-in acceptance | 17 merged PRs; `progress.md` §47 (search hand-off) and §50–57 (18–27 Nov) |
| 8 (to 28 Nov) | Public views, sign-up, registrations, squads, venues, serverless functions, rebuilt images, documentation | `progress.md` §58–60; [test-execution-report.md](test-execution-report.md) |

Retrospective notes still need the team's input: a draft built from the records is in [lessons-learned.md](lessons-learned.md) for confirmation at the final retrospective.
