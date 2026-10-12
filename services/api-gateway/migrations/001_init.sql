CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS dzongkhags (
  name text PRIMARY KEY
);
INSERT INTO dzongkhags(name) VALUES
  ('Bumthang'),('Chhukha'),('Dagana'),('Gasa'),('Haa'),('Lhuntse'),('Mongar'),('Paro'),
  ('Pema Gatshel'),('Punakha'),('Samdrup Jongkhar'),('Samtse'),('Sarpang'),('Thimphu'),
  ('Trashigang'),('Trashiyangtse'),('Trongsa'),('Tsirang'),('Wangdue Phodrang'),('Zhemgang')
ON CONFLICT (name) DO NOTHING;
CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), username text UNIQUE NOT NULL, password_hash text NOT NULL,
  role text NOT NULL,
  dzongkhag text, created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
UPDATE users SET role='viewer' WHERE role='dzongkhag_admin';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('super_admin','match_official','viewer'));
ALTER TABLE users DROP COLUMN IF EXISTS dzongkhag;
CREATE TABLE IF NOT EXISTS tournaments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL, sport text NOT NULL,
  season text NOT NULL, status text NOT NULL DEFAULT 'registration_open', venue text NOT NULL DEFAULT '',
  starts_at date, created_by uuid REFERENCES users(id), dzongkhag text REFERENCES dzongkhags(name),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS dzongkhag text REFERENCES dzongkhags(name);
CREATE TABLE IF NOT EXISTS teams (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL, short_code text NOT NULL,
  dzongkhag text NOT NULL, tournament_id uuid REFERENCES tournaments(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(short_code, tournament_id)
);
CREATE TABLE IF NOT EXISTS matches (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tournament_id uuid NOT NULL REFERENCES tournaments(id),
  home_team_id uuid NOT NULL REFERENCES teams(id), away_team_id uuid NOT NULL REFERENCES teams(id),
  venue text NOT NULL DEFAULT '', kickoff timestamptz, status text NOT NULL DEFAULT 'scheduled',
  live_home integer, live_away integer, official_id uuid REFERENCES users(id),
  result_home integer, result_away integer, finalized_at timestamptz,
  CHECK (home_team_id <> away_team_id), CHECK ((result_home IS NULL) = (finalized_at IS NULL))
);
CREATE TABLE IF NOT EXISTS standings (
  tournament_id uuid NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
  team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  played integer NOT NULL DEFAULT 0, won integer NOT NULL DEFAULT 0, drawn integer NOT NULL DEFAULT 0,
  lost integer NOT NULL DEFAULT 0, goals_for integer NOT NULL DEFAULT 0, goals_against integer NOT NULL DEFAULT 0,
  points integer NOT NULL DEFAULT 0, PRIMARY KEY(tournament_id,team_id)
);
CREATE TABLE IF NOT EXISTS event_outbox (
  id bigserial PRIMARY KEY, subject text NOT NULL, payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz
);
CREATE INDEX IF NOT EXISTS matches_tournament_kickoff ON matches(tournament_id,kickoff);
CREATE INDEX IF NOT EXISTS teams_dzongkhag ON teams(dzongkhag);
CREATE OR REPLACE FUNCTION protect_finalized_match_result() RETURNS trigger AS $$
BEGIN
  IF OLD.finalized_at IS NOT NULL AND (
    NEW.result_home IS DISTINCT FROM OLD.result_home OR
    NEW.result_away IS DISTINCT FROM OLD.result_away OR
    NEW.finalized_at IS DISTINCT FROM OLD.finalized_at
  ) THEN
    RAISE EXCEPTION 'finalized match results are immutable';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS matches_result_immutable ON matches;
CREATE TRIGGER matches_result_immutable BEFORE UPDATE ON matches
FOR EACH ROW EXECUTE FUNCTION protect_finalized_match_result();