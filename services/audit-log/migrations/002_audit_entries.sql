DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'audit_entries' AND column_name = 'actor_id' AND data_type <> 'text'
    ) THEN
        ALTER TABLE audit_entries ALTER COLUMN actor_id TYPE text USING actor_id::text;
    END IF;
END $$;
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS event_id uuid;
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS recorded_at timestamptz NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS audit_entries_event_id_unique
    ON audit_entries(event_id) WHERE event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS audit_entries_resource_occurred_idx
    ON audit_entries(resource_type, resource_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS audit_entries_occurred_idx
    ON audit_entries(occurred_at DESC, id DESC);
