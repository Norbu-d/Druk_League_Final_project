# Search index and synchronization design

Status: implementation contract for SCRUM-87. PostgreSQL remains the source of truth; the Search service owns a rebuildable read model and never writes to source-service databases.

## Indexed entities

Index `team`, `player`, and `tournament` records. Each record is one row in `search_documents`, keyed by `(entity_type, entity_id)`, with the display title/subtitle and normalized searchable text in `content`. Store the source record's monotonically increasing version (or `updated_at` when a version is unavailable) and the last applied event ID alongside the document. Search results are hints for discovery; reads and authorization must be checked against the owning service.

## Data flow

Source services publish versioned upsert and delete events through their transactional outbox after a successful database mutation. A message contains `event_id`, `entity_type`, `entity_id`, `version`, `operation`, `occurred_at`, and the fields needed to build the document. Search consumes the entity subjects using durable JetStream consumers and applies each event in a transaction: reject an event older than the indexed version, deduplicate by event ID, upsert or delete the document, and record the event ID. Acknowledge only after commit. Transient database errors are retried; poison events go to the shared DLQ for inspection/replay.

### Event envelope implemented by Search

Publish to `drukleague.search.team`, `drukleague.search.player`, or `drukleague.search.tournament` according to `entity_type`. The JSON envelope is:

```json
{
  "event_id": "uuid",
  "entity_type": "team",
  "entity_id": "uuid",
  "version": 1,
  "operation": "upsert",
  "occurred_at": "2026-11-19T10:30:00Z",
  "fields": {
    "title": "Thimphu FC",
    "subtitle": "Thimphu · THF",
    "search_text": "Thimphu FC THF Thimphu"
  }
}
```

`event_id` and `entity_id` are UUIDs, `version` is a positive integer that increases for each entity mutation, and `occurred_at` is RFC 3339. `operation` is `upsert` or `delete`; deletes omit `fields`. Upserts require a non-empty `fields.title`; `subtitle` and `search_text` may be empty, in which case Search falls back to title and subtitle. Search records processed event IDs and per-entity versions transactionally. A delete retains its version tombstone so an older upsert cannot restore the removed document.

## Rebuild and consistency

The index is eventually consistent. Team, Player, and Tournament migrations create a service-local `search_outbox`, seed one initial upsert event per existing row, and install database triggers that write versioned upsert/delete events in the same transaction as every later mutation. Each source service relays its outbox to JetStream at least once; Search acknowledges only after committing the event and deduplicates by event ID. Outbox rows are retained, so they are the durable snapshot and change history for repair. To rebuild, stop Team, Player, Tournament, and Search; clear Search's `search_documents`, `search_entity_versions`, and `search_processed_events`; reset each source outbox's `published_at` to NULL; then start Search and the three sources. They replay every retained event in order per entity; Search's version checks make duplicate and out-of-order delivery safe. Do not prune source outboxes unless a replacement snapshot/export path is introduced. This single-generation procedure briefly makes search return no results while replay runs; source data is preserved, while service writes are paused during restart.

## Query behavior and operations

The current gRPC query accepts entity filters and bounded offset pagination. Queries are case-insensitive and limited to 200 characters; page size defaults to 20 and caps at 100. Results sort deterministically by update time, entity type, and ID. Replace substring matching with PostgreSQL full-text/trigram indexing when relevance requirements are agreed; this is an internal storage change and does not change the RPC contract. Metrics should cover consumer lag, retries/DLQ count, indexed documents per type, rebuild duration, and query latency. Search downtime or index loss must not affect source data.

## Rollout dependencies

The Search consumers and Team/Player/Tournament transactional outbox producers are implemented. The initial migration backfills existing rows into retained outboxes; later inserts, updates, and deletes are captured by database triggers. Apply schema changes through the normal service migration runner. Search population is asynchronous and depends on NATS being available; pending outbox rows are retried after service restarts.
