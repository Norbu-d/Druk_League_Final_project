# SWE302 — Defect Log

This log records known defects and delivery gaps from repository review and the runtime evidence recorded through 9 Nov 2026. “Open” means the workflow is not yet implemented or acceptance is not yet verified; it does not imply a production incident.

| ID | Summary | Severity | Status | Reproduction / evidence | Planned resolution |
|---|---|---:|---|---|---|
| D-001 | Dashboard and several screens still display illustrative values after API sign-in | Medium | Open | Sign in and compare overview, officials, venues and reports with persisted records. The app currently refreshes only teams, tournaments, matches and standings. | Add gateway endpoints, replace fixture data, and show loading, error and empty states. |
| D-002 | Media upload and binary retrieval are not implemented | Medium | Open | Media has metadata migrations and a service health shell; there is no registered media RPC handler or binary storage adapter. | Implement authorized upload/retrieval, durable object storage, metadata validation and cleanup. |
| D-003 | Full-stack build and all service acceptance flows are not verified on the current revision | High | Open | Compose services were started; targeted Configuration and Standings images were built. Nginx config validation and an HTTP 200 with security headers were observed. A broad all-service rebuild and full end-to-end run were not recorded for these commits. | Build one image at a time, run targeted acceptance scenarios, and record commit/environment/results. |
| D-004 | Several service contracts remain unwired or incomplete, and cross-service data ownership is inconsistent | High | Open | Analytics match-event aggregation is implemented; media still has no RPC handler. Team stores `tournament_id` while Registration also links teams and tournaments. | Complete service adapters and agree/implement the Registration ownership model; keep event consumers idempotent. |
| D-005 | Fault recovery, dead-letter replay, and admin result-correction audit are not verified end to end | High | Open | DLQ inspect reported zero messages; replay could not be exercised without a safe DLQ entry. Result correction publishes `drukleague.admin.match.result.corrected`, but no end-to-end consumer check is recorded here. | Run controlled Docker fault/recovery and audit-consumer scenarios; record evidence and retain clean state. |

## Triage policy

P1 blocks a demo or risks unauthorized/corrupt results. P2 affects a major workflow or has a workaround. P3 is cosmetic. Resolve an item only after the reproduction path passes on a recorded revision; retain any external or design dependency as open.
