CREATE TABLE IF NOT EXISTS configuration_entries (key text PRIMARY KEY, value text NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
