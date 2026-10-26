ALTER TABLE standings ADD COLUMN IF NOT EXISTS team_name text NOT NULL DEFAULT '';
ALTER TABLE standings ADD COLUMN IF NOT EXISTS dzongkhag text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS processed_events (
  event_id uuid PRIMARY KEY,
  processed_at timestamptz NOT NULL DEFAULT now()
);
