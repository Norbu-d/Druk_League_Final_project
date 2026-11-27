ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS format text NOT NULL DEFAULT 'round_robin';
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS bracket_match integer;
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS home_seed integer;
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS away_seed integer;

CREATE UNIQUE INDEX IF NOT EXISTS fixtures_bracket_match_unique
  ON fixtures(tournament_id, round, bracket_match) WHERE bracket_match IS NOT NULL;

CREATE TABLE IF NOT EXISTS fixture_bracket_state (
  tournament_id uuid PRIMARY KEY,
  bracket_size integer NOT NULL,
  total_rounds integer NOT NULL
);

CREATE TABLE IF NOT EXISTS fixture_bracket_entries (
  tournament_id uuid NOT NULL,
  round integer NOT NULL,
  match_no integer NOT NULL,
  slot_no integer NOT NULL CHECK (slot_no IN (1,2)),
  team_id uuid NOT NULL,
  seed integer NOT NULL,
  PRIMARY KEY (tournament_id, round, match_no, slot_no),
  UNIQUE (tournament_id, round, team_id)
);

CREATE TABLE IF NOT EXISTS fixture_match_results (
  event_id uuid PRIMARY KEY,
  match_id uuid NOT NULL UNIQUE,
  tournament_id uuid NOT NULL,
  home_team_id uuid NOT NULL,
  away_team_id uuid NOT NULL,
  home_score integer NOT NULL,
  away_score integer NOT NULL,
  finalized_at timestamptz NOT NULL
);
