ALTER TABLE teams ADD COLUMN IF NOT EXISTS manager_id uuid;
CREATE INDEX IF NOT EXISTS teams_manager_id_idx ON teams(manager_id) WHERE manager_id IS NOT NULL;
