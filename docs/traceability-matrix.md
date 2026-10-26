# SWE302 — Requirements Traceability Matrix

| Requirement ID | Requirement | Design / implementation location | Verification case | Status |
|---|---|---|---|---|
| R-01 | Users authenticate before API access | `services/api-gateway/cmd/api/main.go` login/JWT middleware | T-01, T-02 | Implemented; execution pending |
| R-02 | System distinguishes tournament admin, match official, and viewer | users table, JWT principal, API handlers | T-03, T-04, T-13 | Implemented; execution pending |
| R-03 | Teams and tournaments carry a dzongkhag reference; there is no dzongkhag-scoped admin role | `dzongkhags` reference table, team and tournament fields | T-03, T-13 | Implemented in MVP schema/API; existing tournament records need district values |
| R-04 | Admin can create tournaments and register teams | `POST /api/tournaments`, `POST /api/teams` | T-15 | Implemented; UI/API execution pending |
| R-05 | Fixture generation prevents duplicate generation and same-time clashes | match creation and tournament fixture route | T-08, T-09, T-10, T-11 | MVP implemented; venue-duration policy pending |
| R-06 | Live scores are mutable but do not affect standings | Score `UpdateLiveScore` / `GetScore`; Standings consumer | T-05 | Score RPC verified; pre-finalization standings check pending |
| R-07 | Only assigned match official can finalize once | Match assignment; Score reads Match.GetMatch and checks caller ID; immutable score repository/trigger | T-04, T-06 | Official update/finalize verified; admin denied; post-final mutation rejected |
| R-08 | Final results update standings once and cannot be edited | Score result/outbox transaction; Score immutability trigger; Standings consumer transaction with `event_id` dedupe | T-06, T-07 | 2–1 event and duplicate replay verified; other result permutations pending |
| R-09 | MatchFinalized is durably delivered to NATS and consumers apply it idempotently | Score `event_outbox` publisher; Match completion and Standings JetStream consumers | T-07, T-12 | First live event and duplicate handling verified; outage/reconnect and dead-letter pending |
| R-10 | Teams, fixtures, and standings can be read through API | `/api/teams`, `/api/matches`, `/api/standings` | T-13 | Implemented; UI binding partial |
| R-11 | Twenty gRPC services use versioned protobuf contracts | proto/drukleague, packages/contracts/gen, services/* | Buf lint/build and per-service RPC integration | Contracts generated; 9 services have verified domain handlers, 11 remain shells |
| R-12 | System demonstrates resilience and traces | planned service circuit breaker and OpenTelemetry | Fault-injection scenarios | Not implemented |
| R-13 | Reproducible local deployment | `deploy/docker-compose.yml`, Dockerfiles, Nginx | Compose start/health/e2e | Core Compose stack running; Match, Score and Standings rebuilt and verified in containers |
| R-14 | Quality evidence and project-management documents accompany submission | `docs/` | Review artifacts and executed test records | Documents in progress; test execution pending |
