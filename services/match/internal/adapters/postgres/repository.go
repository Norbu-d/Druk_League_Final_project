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
	if err = lockSchedule(ctx, tx); err != nil {
		return domain.Match{}, err
	}
	// A team or venue is busy for SlotLength either side of any of its kickoffs.
	var teamClash bool
	var at time.Time
	err = tx.QueryRowContext(ctx, `SELECT home_team_id IN ($3,$4) OR away_team_id IN ($3,$4), kickoff FROM matches
		WHERE kickoff > $1 AND kickoff < $2 AND (home_team_id IN ($3,$4) OR away_team_id IN ($3,$4) OR ($5<>'' AND lower(venue)=lower($5)))
		ORDER BY kickoff LIMIT 1`, m.Kickoff.Add(-domain.SlotLength), m.Kickoff.Add(domain.SlotLength), m.HomeTeamID, m.AwayTeamID, m.Venue).Scan(&teamClash, &at)
	switch {
	case err == nil && teamClash:
		return domain.Match{}, domain.Clash("a team", at)
	case err == nil:
		return domain.Match{}, domain.Clash("venue "+m.Venue, at)
	case !errors.Is(err, sql.ErrNoRows):
		return domain.Match{}, err
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

// AssignOfficial sets the match official unless the match is completed or the official
// already has another unfinished match within SlotLength of this kickoff.
func (r *Repository) AssignOfficial(ctx context.Context, id, officialID string) (domain.Match, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Match{}, err
	}
	defer tx.Rollback()
	if err = lockSchedule(ctx, tx); err != nil {
		return domain.Match{}, err
	}
	var kickoff sql.NullTime
	var current string
	err = tx.QueryRowContext(ctx, `SELECT kickoff,status FROM matches WHERE id=$1`, id).Scan(&kickoff, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Match{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Match{}, err
	}
	if current == domain.StatusCompleted {
		return domain.Match{}, domain.ErrConflict
	}
	if kickoff.Valid {
		var at time.Time
		err = tx.QueryRowContext(ctx, `SELECT kickoff FROM matches WHERE official_id=$1 AND id<>$2 AND status<>$3 AND kickoff > $4 AND kickoff < $5 ORDER BY kickoff LIMIT 1`,
			officialID, id, domain.StatusCompleted, kickoff.Time.Add(-domain.SlotLength), kickoff.Time.Add(domain.SlotLength)).Scan(&at)
		if err == nil {
			return domain.Match{}, domain.Clash("the official", at)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.Match{}, err
		}
	}
	m, err := scan(tx.QueryRowContext(ctx, `UPDATE matches SET official_id=$2 WHERE id=$1 RETURNING id::text,tournament_id::text,home_team_id::text,away_team_id::text,venue,kickoff,status,COALESCE(official_id::text,'')`, id, officialID))
	if err != nil {
		return domain.Match{}, err
	}
	return m, tx.Commit()
}

// lockSchedule serialises every booking change, so two concurrent requests cannot both pass
// the overlap checks for the same team, venue or official.
func lockSchedule(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('match-schedule'))`)
	return err
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
