# Architecture decision: DrukLeague 20-service target

**Status:** Accepted target; implementation is staged.

The current proposal defines 20 independently deployable Go microservices. The running product began as a single Go HTTP API MVP and is not the final target architecture. On 7 Oct 2026, the project owner confirmed that the team is building toward all 20 services.

The target service boundaries use ports and adapters: domain and application code depend on ports; PostgreSQL, gRPC and NATS adapters implement those ports. Each service owns a database and migration set. The API gateway and BFF will route browser requests to service gRPC APIs. The current code retains the shared PostgreSQL MVP API and has 20 service modules. Nine services have verified domain handlers; Match, Score and Standings now use hexagonal layers and their own databases. Gateway/BFF routing and frontend integration remain pending.

In the monolith MVP, finalization commits the result, standings rows and outbox event in one transaction. Across the split Score and Standings databases, no such transaction is possible: Score atomically commits the immutable result and outbox event, and publishes `MatchFinalized` to JetStream with `event_id`. Match consumes the event to mark its match completed. Standings consumes it asynchronously, looks up team display data on first encounter, and applies both rows together with its `event_id` dedupe record in one local transaction. Manual container verification confirmed the flow and that replaying the same event does not double-count standings. Broader consumers, NATS outage/reconnect behavior, and dead-letter handling remain future work.

Final results cannot be edited in place in the MVP. The correction policy remains open between an audited admin correction with reason and append-only correction events followed by standings recalculation.

Fixture clash validation rejects another match at the same exact kickoff timestamp when a team or non-empty venue overlaps. Venue availability windows and configurable match durations remain future work.

The current default bootstrap user and secrets are for local development only. Deployment must override them with unique secrets.
