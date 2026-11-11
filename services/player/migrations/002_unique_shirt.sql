CREATE INDEX IF NOT EXISTS players_team_idx ON players(team_id);

-- One shirt number per team. Skipped (with a notice) if old rows already clash, so the
-- service still starts; clean the duplicates up and restart to add the index.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM players WHERE shirt_number <> '' GROUP BY team_id, shirt_number HAVING count(*) > 1) THEN
    CREATE UNIQUE INDEX IF NOT EXISTS players_team_shirt_unique ON players (team_id, shirt_number) WHERE shirt_number <> '';
  ELSE
    RAISE NOTICE 'players_team_shirt_unique not created: duplicate shirt numbers exist';
  END IF;
END $$;
