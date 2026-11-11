# SWE302 — Defect Log

This log records known defects and delivery gaps from repository review and the runtime evidence recorded through 9 Nov 2026. “Open” means the workflow is not yet implemented or acceptance is not yet verified; it does not imply a production incident.

| ID | Summary | Severity | Status | Reproduction / evidence | Planned resolution |
|---|---|---:|---|---|---|
| D-001 | Dashboard and several screens still display illustrative values after API sign-in | Medium | Open | Frontend API refresh covers teams, tournaments, matches and standings; dashboard aggregates, officials, venues and other screens still use sample values. | Add API-backed views and loading, error, and empty states. |
| D-002 | Media upload and binary retrieval lack complete acceptance evidence | Medium | In Progress | Media now implements authenticated upload, per-match listing, retrieval and owner/admin deletion with persistent blob storage. Upload/download/delete browser E2E and restore-from-backup have not been checked. | Run authenticated Compose/browser workflow and verify backup/restore behavior. |
| D-003 | Full-stack and current-revision acceptance is incomplete | High | Open | GitHub CI checks for `fdd5ce7` passed 8/8. Docker-backed full Compose and browser E2E have not been recorded on this revision. | Run the full Compose smoke flow and record commit, environment, results and defects. |
| D-004 | Some service contracts and cross-service ownership still need completion | High | Open | Analytics and Configuration handlers, Media HTTP APIs, and Audit Log consumers exist. Analytics/Configuration runtime RPC acceptance, Search indexing and Team/Registration product rules remain open. | Complete targeted runtime acceptance and settle the remaining ownership rules with service owners. |
| D-005 | Fault recovery and dead-letter replay lack live end-to-end evidence | High | Open | EventKit redelivery/DLQ logic has embedded JetStream tests, and the event admin replay command exists. A controlled Docker outage/reconnect and replay of a safe DLQ entry are not recorded. | Run controlled Docker recovery/replay scenarios; preserve the replay source record and capture evidence. |

## Triage policy

P1 blocks a demo or risks unauthorized/corrupt results. P2 affects a major workflow or has a workaround. P3 is cosmetic. Resolve an item only after the reproduction path passes on a recorded revision; retain any external or design dependency as open.
