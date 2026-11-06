# Search index and synchronization design

Status: implementation contract for SCRUM-87. PostgreSQL remains the source of truth; the Search service owns a rebuildable read model and never writes to source-service databases.

## Indexed entities

Index `team`, `player`, and `tournament` records. Each record is one row in `search_documents`, keyed by `(entity_type, entity_id)`, with the display title/subtitle and normalized searchable text in `content`. Store the source record's monotonically increasing version (or `updated_at` when a version is unavailable) and the last applied event ID alongside the document. Search results are hints for discovery; reads and authorization must be checked against the owning service.

## Data flow

Source services publish versioned upsert and delete events through their transactional outbox after a successful database mutation. A message contains `event_id`, `entity_type`, `entity_id`, `version`, `operation`, `occurred_at`, and the fields needed to build the document. Search consumes the entity subjects using durable JetStream consumers and applies each event in a transaction: reject an event older than the indexed version, deduplicate by event ID, upsert or delete the document, and record the event ID. Acknowledge only after commit. Transient database errors are retried; poison events go to the shared DLQ for inspection/replay.

## Rebuild and consistency

The index is eventually consistent. Event delivery is at least once, so consumers must be idempotent. A rebuild reads a consistent snapshot from each source service's export/list API, writes documents to a new generation, then starts/replays events from a captured stream sequence before switching the active generation. This avoids a gap between snapshot and live updates. Search remains available against the previous generation while rebuilding. Deletes are represented by tombstones/events so stale documents do not return after replay. If an entity's event history expires, the source snapshot repairs the index.

## Query behavior and operations

The current gRPC query accepts entity filters and bounded offset pagination. Queries are case-insensitive and limited to 200 characters; page size defaults to 20 and caps at 100. Results sort deterministically by update time, entity type, and ID. Replace substring matching with PostgreSQL full-text/trigram indexing when relevance requirements are agreed; this is an internal storage change and does not change the RPC contract. Metrics should cover consumer lag, retries/DLQ count, indexed documents per type, rebuild duration, and query latency. Search downtime or index loss must not affect source data.

## Rollout dependencies

Each owner service must add outbox events for its entity mutations and expose a consistent snapshot/list path. Until those producers and snapshot APIs exist, the current Search RPC can query only documents inserted by an operator or future consumer; a successful service build does not mean the index is populated. Schema changes should be applied with the normal service migration runner.
