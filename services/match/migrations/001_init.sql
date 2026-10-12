CREATE TABLE IF NOT EXISTS matches (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tournament_id uuid NOT NULL, home_team_id uuid NOT NULL, away_team_id uuid NOT NULL, venue_id uuid, kickoff timestamptz, status text NOT NULL DEFAULT 'scheduled', official_id uuid);

ALTER TABLE matches ADD COLUMN IF NOT EXISTS venue text NOT NULL DEFAULT '';
