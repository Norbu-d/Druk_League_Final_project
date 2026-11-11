-- One tournament name per season (case-insensitive). Skipped (with a notice) if old rows
-- already clash, so the service still starts; clean the duplicates up and restart to add it.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM tournaments GROUP BY lower(name), season HAVING count(*) > 1) THEN
    CREATE UNIQUE INDEX IF NOT EXISTS tournaments_name_season_unique ON tournaments (lower(name), season);
  ELSE
    RAISE NOTICE 'tournaments_name_season_unique not created: duplicate names exist';
  END IF;
END $$;
