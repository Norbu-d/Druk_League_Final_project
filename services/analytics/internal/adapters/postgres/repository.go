package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) RecordFinalized(ctx context.Context, event domain.MatchFinalized, tournament domain.Tournament) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_match_facts (
			event_id, match_id, tournament_id, home_team_id, away_team_id,
			home_score, away_score, season, dzongkhag, finalized_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (event_id) DO NOTHING`,
		event.EventID, event.MatchID, event.TournamentID, event.HomeTeamID, event.AwayTeamID,
		event.HomeScore, event.AwayScore, tournament.Season, tournament.Dzongkhag, event.FinalizedAt)
	return err
}
