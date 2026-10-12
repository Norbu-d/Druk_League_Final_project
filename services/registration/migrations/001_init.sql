CREATE TABLE IF NOT EXISTS registrations (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tournament_id uuid NOT NULL, team_id uuid NOT NULL, status text NOT NULL DEFAULT 'pending', submitted_at timestamptz NOT NULL DEFAULT now(), UNIQUE(tournament_id,team_id));

ALTER TABLE registrations ADD COLUMN IF NOT EXISTS review_reason text NOT NULL DEFAULT '';
ALTER TABLE registrations ADD COLUMN IF NOT EXISTS reviewed_at timestamptz;
