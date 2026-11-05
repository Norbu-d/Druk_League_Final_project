-- Store finalized match facts for idempotent tournament analytics.
-- Tournament metadata is copied in by the consumer so dashboard filters do
-- not require cross-service joins at query time.
CREATE TABLE IF NOT EXISTS analytics_match_facts (
    event_id uuid PRIMARY KEY,
    match_id uuid NOT NULL UNIQUE,
    tournament_id uuid NOT NULL,
    home_team_id uuid NOT NULL,
    away_team_id uuid NOT NULL,
    home_score integer NOT NULL CHECK (home_score BETWEEN 0 AND 99),
    away_score integer NOT NULL CHECK (away_score BETWEEN 0 AND 99),
    season text NOT NULL,
    dzongkhag text NOT NULL,
    finalized_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT analytics_match_facts_distinct_teams
        CHECK (home_team_id <> away_team_id)
);

CREATE INDEX IF NOT EXISTS analytics_match_facts_tournament_finalized_idx
    ON analytics_match_facts (tournament_id, finalized_at DESC);

CREATE INDEX IF NOT EXISTS analytics_match_facts_season_dzongkhag_idx
    ON analytics_match_facts (season, dzongkhag, finalized_at DESC);

CREATE INDEX IF NOT EXISTS analytics_match_facts_home_team_idx
    ON analytics_match_facts (home_team_id);

CREATE INDEX IF NOT EXISTS analytics_match_facts_away_team_idx
    ON analytics_match_facts (away_team_id);
