ALTER TABLE search_documents ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE search_documents ADD COLUMN IF NOT EXISTS last_event_id text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS search_entity_versions (
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    version bigint NOT NULL,
    deleted boolean NOT NULL DEFAULT false,
    last_event_id text NOT NULL,
    PRIMARY KEY(entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS search_processed_events (
    event_id text PRIMARY KEY,
    processed_at timestamptz NOT NULL DEFAULT now()
);
