CREATE TABLE IF NOT EXISTS event_outbox (
  event_id uuid PRIMARY KEY,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
CREATE INDEX IF NOT EXISTS event_outbox_unpublished_idx ON event_outbox(created_at) WHERE published_at IS NULL;
