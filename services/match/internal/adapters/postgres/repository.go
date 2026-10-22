package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/drukleague/platform/services/match/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, m domain.Match) (domain.Match, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Match{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, m.Kickoff.UTC().Format(time.RFC3339Nano)); err != nil {
		return domain.Match{}, err
	}
	var clash bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE kickoff=$1 AND (home_team_id IN ($2,$3) OR away_team_id IN ($2,$3) OR ($4<>'' AND venue=$4)))`, m.Kickoff, m.HomeTeamID, m.AwayTeamID, m.Venue).Scan(&clash)
	if err != nil {
		return domain.Match{}, err
	}
	if clash {
		return domain.Match{}, domain.ErrConflict
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO matches(tournament_id,home_team_id,away_team_id,venue,kickoff,status) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'')`, m.TournamentID, m.HomeTeamID, m.AwayTeamID, m.Venue, m.Kickoff, domain.StatusScheduled).Scan(&m.ID, &m.TournamentID, &m.HomeTeamID, &m.AwayTeamID, &m.Venue, &m.Kickoff, &m.Status, &m.OfficialID)
	if err != nil {
		return domain.Match{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Match{}, err
	}
	return m, nil
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Match, error) {
	m, err := scan(r.db.QueryRowContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'') FROM matches WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Match{}, domain.ErrNotFound
	}
	return m, err
}

func (r *Repository) List(ctx context.Context, tournamentID string, after *time.Time, limit, offset int) ([]domain.Match, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'') FROM matches WHERE ($1='' OR tournament_id::text=$1) AND ($2::timestamptz IS NULL OR kickoff >= $2) ORDER BY kickoff,id LIMIT $3 OFFSET $4`, tournamentID, after, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Match, 0, limit)
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		if len(items) == limit {
			return items, true, nil
		}
		items = append(items, m)
	}
	return items, false, rows.Err()
}

func (r *Repository) AssignOfficial(ctx context.Context, id, officialID string) (domain.Match, error) {
	m, err := scan(r.db.QueryRowContext(ctx, `UPDATE matches SET official_id=$2 WHERE id=$1 AND status<>$3 RETURNING id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'')`, id, officialID, domain.StatusCompleted))
	if errors.Is(err, sql.ErrNoRows) {
		current, getErr := r.Get(ctx, id)
		if errors.Is(getErr, domain.ErrNotFound) {
			return domain.Match{}, domain.ErrNotFound
		}
		if getErr != nil {
			return domain.Match{}, getErr
		}
		if current.Status == domain.StatusCompleted {
			return domain.Match{}, domain.ErrConflict
		}
	}
	return m, err
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Match, error) {
	var m domain.Match
	var kickoff sql.NullTime
	err := row.Scan(&m.ID, &m.TournamentID, &m.HomeTeamID, &m.AwayTeamID, &m.Venue, &kickoff, &m.Status, &m.OfficialID)
	if err != nil {
		return m, err
	}
	if kickoff.Valid {
		m.Kickoff = kickoff.Time
	}
	return m, nil
}

func IsInvalidUUID(err error) bool {
	return err != nil && strings.Contains(err.Error(), "invalid input syntax for type uuid")
}

// MarkCompleted records the event id and completes the match in one transaction, so a
// redelivered MatchFinalized event is applied at most once.
func (r *Repository) MarkCompleted(ctx context.Context, eventID, matchID string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO processed_events(event_id) VALUES($1) ON CONFLICT(event_id) DO NOTHING`, eventID)
	if err != nil {
		return false, err
	}
	if inserted, err := result.RowsAffected(); err != nil || inserted == 0 {
		return false, err
	}
	result, err = tx.ExecContext(ctx, `UPDATE matches SET status=$2 WHERE id=$1`, matchID, domain.StatusCompleted)
	if err != nil {
		return false, err
	}
	if updated, err := result.RowsAffected(); err != nil || updated == 0 {
		if err == nil {
			err = domain.ErrNotFound
		}
		return false, err
	}
	return true, tx.Commit()
}
