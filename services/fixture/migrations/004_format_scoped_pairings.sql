-- Teams may meet once in the league stage and again in the knockout bracket.
-- Keep duplicate prevention within a format and round, not across both stages.
DROP INDEX IF EXISTS fixtures_pairing_unique;

CREATE UNIQUE INDEX IF NOT EXISTS fixtures_pairing_unique
  ON fixtures(tournament_id, format, round, home_team_id, away_team_id);
