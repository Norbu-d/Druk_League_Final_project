# SWE302 — Requirements Traceability Matrix

| Requirement ID | Requirement | Design / implementation location | Verification case | Status |
|---|---|---|---|---|
| R-01 | Users authenticate before API access | `services/api-gateway/cmd/api/main.go` login/JWT middleware | T-01, T-02 | Implemented; execution pending |
| R-02 | System distinguishes tournament admin, match official, and viewer | users table, JWT principal, API handlers | T-03, T-04, T-13 | Implemented; execution pending |
| R-03 | Teams and tournaments carry a dzongkhag reference; there is no dzongkhag-scoped admin role | `dzongkhags` reference table, team and tournament fields | T-03, T-13 | Implemented in MVP schema/API; existing tournament records need district values |
| R-04 | Admin can create tournaments and register teams | `POST /api/tournaments`, `POST /api/teams` | T-15 | Implemented; UI/API execution pending |
| R-05 | Fixture generation prevents duplicate generation and same-time clashes | match creation and tournament fixture route | T-08, T-09, T-10, T-11 | MVP implemented; venue-duration policy pending |
| R-06 | Live scores are mutable but do not affect standings | `PATCH /api/matches/{id}/live` | T-05 | Implemented; execution pending |
| R-07 | Only assigned match official can finalize once | official assignment and finalize transaction | T-04, T-06 | Implemented; execution pending |
| R-08 | Final results update standings once and cannot be edited | transaction, outbox, immutable-result trigger | T-06, T-07 | Implemented; execution pending |
| R-09 | MatchFinalized is durably delivered to NATS | `event_outbox`, JetStream publisher | T-12 | Publisher implemented; consumer pending |
| R-10 | Teams, fixtures, and standings can be read through API | `/api/teams`, `/api/matches`, `/api/standings` | T-13 | Implemented; UI binding partial |
| R-11 | Twenty gRPC services use versioned protobuf contracts | proto/drukleague, packages/contracts/gen, services/* | Buf lint/build and per-service RPC integration | Contracts generated; domain APIs pending |
| R-12 | System demonstrates resilience and traces | planned service circuit breaker and OpenTelemetry | Fault-injection scenarios | Not implemented |
| R-13 | Reproducible local deployment | `deploy/docker-compose.yml`, Dockerfiles, Nginx | Compose start/health/e2e | Config authored; container runtime unverified |
| R-14 | Quality evidence and project-management documents accompany submission | `docs/` | Review artifacts and executed test records | Documents in progress; test execution pending |
