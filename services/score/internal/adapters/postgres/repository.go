package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/drukleague/platform/services/score/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) UpdateLive(ctx context.Context, matchID string, home, away int32) (domain.Score, error) {
	var score domain.Score
	err := r.db.QueryRowContext(ctx, `INSERT INTO scores(match_id,home_score,away_score,updated_at) VALUES($1,$2,$3,now()) ON CONFLICT(match_id) DO UPDATE SET home_score=EXCLUDED.home_score,away_score=EXCLUDED.away_score,updated_at=now() WHERE scores.finalized_at IS NULL RETURNING match_id::text,home_score,away_score,finalized_at IS NOT NULL,updated_at`, matchID, home, away).Scan(&score.MatchID, &score.HomeScore, &score.AwayScore, &score.Finalized, &score.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Score{}, domain.ErrConflict
	}
	return score, err
}

// Finalize locks the score and writes the outbox event atomically (transactional outbox).
func (r *Repository) Finalize(ctx context.Context, match domain.Match, eventID, finalizedBy, reason string) (domain.Score, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Score{}, err
	}
	defer tx.Rollback()
	var score domain.Score
	var finalizedAt time.Time
	err = tx.QueryRowContext(ctx, `INSERT INTO scores(match_id,home_score,away_score,finalized_at,updated_at) VALUES($1,0,0,now(),now()) ON CONFLICT(match_id) DO UPDATE SET finalized_at=now(),updated_at=now() WHERE scores.finalized_at IS NULL RETURNING match_id::text,home_score,away_score,finalized_at,updated_at`, match.ID).Scan(&score.MatchID, &score.HomeScore, &score.AwayScore, &finalizedAt, &score.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Score{}, domain.ErrConflict
	}
	if err != nil {
		return domain.Score{}, err
	}
	score.Finalized = true
	payload, err := json.Marshal(domain.MatchFinalized{EventID: eventID, MatchID: match.ID, TournamentID: match.TournamentID, HomeTeamID: match.HomeTeamID, AwayTeamID: match.AwayTeamID, HomeScore: score.HomeScore, AwayScore: score.AwayScore, FinalizedAt: finalizedAt.UTC(), FinalizedBy: finalizedBy, Reason: reason})
	if err != nil {
		return domain.Score{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(event_id,payload) VALUES($1,$2::jsonb)`, eventID, string(payload)); err != nil {
		return domain.Score{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Score{}, err
	}
	return score, nil
}

func (r *Repository) Get(ctx context.Context, matchID string) (domain.Score, error) {
	var score domain.Score
	err := r.db.QueryRowContext(ctx, `SELECT match_id::text,home_score,away_score,finalized_at IS NOT NULL,updated_at FROM scores WHERE match_id=$1`, matchID).Scan(&score.MatchID, &score.HomeScore, &score.AwayScore, &score.Finalized, &score.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Score{}, domain.ErrNotFound
	}
	return score, err
}

// PendingEvents returns unpublished outbox rows, oldest first.
func (r *Repository) PendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id::text,payload::text FROM event_outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]domain.OutboxEvent, 0, limit)
	for rows.Next() {
		var event domain.OutboxEvent
		var payload string
		if err := rows.Scan(&event.EventID, &payload); err != nil {
			return nil, err
		}
		event.Payload = []byte(payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) MarkPublished(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_outbox SET published_at=now() WHERE event_id=$1 AND published_at IS NULL`, eventID)
	return err
}
