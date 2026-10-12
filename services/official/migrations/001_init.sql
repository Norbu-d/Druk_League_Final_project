CREATE TABLE IF NOT EXISTS official_assignments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), match_id uuid NOT NULL, official_user_id uuid NOT NULL, assigned_at timestamptz NOT NULL DEFAULT now(), UNIQUE(match_id,official_user_id));

CREATE UNIQUE INDEX IF NOT EXISTS official_assignments_one_per_match ON official_assignments(match_id);
