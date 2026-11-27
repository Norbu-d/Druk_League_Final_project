-- The overlap checks search by kickoff range and by official.
CREATE INDEX IF NOT EXISTS matches_kickoff_idx ON matches(kickoff);
CREATE INDEX IF NOT EXISTS matches_official_kickoff_idx ON matches(official_id, kickoff) WHERE official_id IS NOT NULL;
