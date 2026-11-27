-- The team manager who owns the team (empty for teams the admin creates), so Registration and
-- Player can check ownership.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS manager_id uuid;
CREATE INDEX IF NOT EXISTS teams_manager_idx ON teams(manager_id);
CREATE INDEX IF NOT EXISTS teams_tournament_idx ON teams(tournament_id);

-- One short code per tournament. Skipped (with a notice) if old rows already clash, so the
-- service still starts; clean the duplicates up and restart to add the index.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM teams WHERE tournament_id IS NOT NULL GROUP BY tournament_id, upper(short_code) HAVING count(*) > 1) THEN
    CREATE UNIQUE INDEX IF NOT EXISTS teams_tournament_code_unique ON teams (tournament_id, upper(short_code));
  ELSE
    RAISE NOTICE 'teams_tournament_code_unique not created: duplicate short codes exist';
  END IF;
END $$;
