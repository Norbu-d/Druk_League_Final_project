-- Assignments keep the match's tournament (copied from the Match service) and who assigned them.
ALTER TABLE official_assignments ADD COLUMN IF NOT EXISTS tournament_id uuid;
ALTER TABLE official_assignments ADD COLUMN IF NOT EXISTS assigned_by uuid;
CREATE INDEX IF NOT EXISTS official_assignments_official_idx ON official_assignments(official_user_id);
CREATE INDEX IF NOT EXISTS official_assignments_tournament_idx ON official_assignments(tournament_id);
