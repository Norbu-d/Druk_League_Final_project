# Service design: hexagonal architecture and resilience

Moved out of `progress.md` on 2026-10-14. This is the target design for each service; `progress.md` holds the current status.

## Hexagonal (ports and adapters), applied per service

Cockburn (2005) is cited in the proposal's literature review; this is how each service should use it.

```text
services/match/
├── internal/domain/        # Match, Score, rules (immutable once finalised)
├── internal/app/           # use cases: SubmitScore, FinaliseMatch
├── internal/ports/         # interfaces: MatchRepo, EventPublisher, TeamLookup
├── internal/adapters/
│   ├── grpc/               # inbound: gRPC handlers
│   ├── postgres/           # outbound: repository + outbox table
│   ├── nats/               # outbound: JetStream publisher
│   └── teamclient/         # outbound: gRPC client to the Team service
└── cmd/match/main.go       # wiring only
```

- The core (domain + app) imports no database, gRPC or NATS packages. Adapters depend on the core, never the reverse.
- Ports are interfaces owned by the core. Inbound adapters call use cases; outbound adapters implement ports.
- Unit tests use fakes or GoMock for the ports (no database or NATS needed), which is what the 80% coverage target relies on. Integration tests (Testcontainers) cover the adapters.
- Hexagonal does not make a system resilient by itself; it tells you where resilience goes: in the **outbound adapters**, so domain code does not change.

## Resilience patterns

| Pattern | Where it lives | Status |
|---|---|---|
| Transactional outbox (result + standings + event in one transaction) | Match's postgres adapter | Implemented in the monolith; carry into the Match service |
| Stable message IDs (publisher de-duplication) | NATS publisher adapter | In the monolith's publisher |
| Durable JetStream, redelivery limit, dead-letter stream | NATS consumer adapters | Planned |
| Idempotent consumers (skip already-processed event IDs) | app layer | Planned |
| gRPC deadlines (timeouts) | outbound client adapters | Planned |
| Retry with backoff, safe reads only | outbound client adapters | Planned |
| Circuit breaker (Match → Team, gateway/BFF → services) | outbound client adapters | Planned; needed for the Sprint 7 demo |
| Cross-service reference check (Team verifies `tournament_id` with Tournament) | Team's outbound client adapter | Gap confirmed 12 Oct 2026; planned with the deadline/retry/breaker work |
| Database per service | each service | Done for the new services; the monolith keeps its own database |
| Health and readiness endpoints | each service | `/healthz` in the API; gRPC health in the shells |
| Graceful degradation (reads keep working if Notification is down) | app layer | Planned |
| OpenTelemetry tracing | all services | Planned |
| Rate limiting | gateway | Not planned |

## Fault-isolation scenarios to test (Sprint 7)

- Stop Team: Match still accepts scores (breaker opens).
- Stop Tournament: Team returns `Unavailable` instead of writing unverified data.
- Stop NATS: outbox rows accumulate and drain on recovery.
- Stop Notification: finalisation still succeeds.
- Deliver a duplicate event: it is processed once.
- Send a poison message: it ends up in the dead-letter stream.