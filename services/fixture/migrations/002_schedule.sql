-- Schedule details, the link to the Match service, and the FixturesGenerated outbox.
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS round integer NOT NULL DEFAULT 1;
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS venue_name text NOT NULL DEFAULT '';
ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS match_id uuid;
CREATE UNIQUE INDEX IF NOT EXISTS fixtures_pairing_unique ON fixtures(tournament_id, round, home_team_id, away_team_id);
CREATE INDEX IF NOT EXISTS fixtures_tournament_kickoff_idx ON fixtures(tournament_id, kickoff);
CREATE TABLE IF NOT EXISTS event_outbox (
  event_id uuid PRIMARY KEY,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
CREATE INDEX IF NOT EXISTS fixture_outbox_unpublished_idx ON event_outbox(created_at) WHERE published_at IS NULL;
