CREATE TABLE IF NOT EXISTS tournaments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL, sport text NOT NULL, season text NOT NULL, dzongkhag text NOT NULL, status text NOT NULL DEFAULT 'registration_open', created_at timestamptz NOT NULL DEFAULT now());

ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS venue text NOT NULL DEFAULT '';
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS starts_at timestamptz;
