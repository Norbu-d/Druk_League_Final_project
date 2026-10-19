package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/fixture/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,tournament_id::text,home_team_id::text,away_team_id::text,COALESCE(venue_id::text,''),venue_name,COALESCE(match_id::text,''),round,kickoff`

// SaveSchedule serialises generation per tournament with an advisory lock so two admins
// pressing "Generate" at once cannot create two schedules.
func (r *Repository) SaveSchedule(ctx context.Context, tournamentID string, fixtures []domain.Fixture, event domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('fixtures:'||$1))`, tournamentID); err != nil {
		return nil, false, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM fixtures WHERE tournament_id=$1`, tournamentID).Scan(&existing); err != nil {
		return nil, false, err
	}
	created := existing == 0
	if created {
		for _, f := range fixtures {
			if _, err = tx.ExecContext(ctx, `INSERT INTO fixtures(tournament_id,home_team_id,away_team_id,venue_id,venue_name,round,kickoff) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7)`,
				f.TournamentID, f.HomeTeamID, f.AwayTeamID, f.VenueID, f.VenueName, f.Round, f.Kickoff); err != nil {
				return nil, false, err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(event_id,payload) VALUES($1,$2::jsonb)`, event.EventID, string(event.Payload)); err != nil {
			return nil, false, err
		}
	}
	stored, err := list(ctx, tx, tournamentID, 10000, 0)
	if err != nil {
		return nil, false, err
	}
	return stored, created, tx.Commit()
}

func (r *Repository) LinkMatch(ctx context.Context, fixtureID, matchID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE fixtures SET match_id=$2 WHERE id=$1 AND match_id IS NULL`, fixtureID, matchID)
	return err
}

func (r *Repository) List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error) {
	items, err := list(ctx, r.db, tournamentID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	if len(items) > limit {
		return items[:limit], true, nil
	}
	return items, false, nil
}

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func list(ctx context.Context, q querier, tournamentID string, limit, offset int) ([]domain.Fixture, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+columns+` FROM fixtures WHERE tournament_id=$1 ORDER BY round,kickoff,id LIMIT $2 OFFSET $3`, tournamentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Fixture{}
	for rows.Next() {
		var f domain.Fixture
		var kickoff sql.NullTime
		if err := rows.Scan(&f.ID, &f.TournamentID, &f.HomeTeamID, &f.AwayTeamID, &f.VenueID, &f.VenueName, &f.MatchID, &f.Round, &kickoff); err != nil {
			return nil, err
		}
		f.Kickoff = kickoff.Time
		items = append(items, f)
	}
	return items, rows.Err()
}

func (r *Repository) PendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id::text,payload::text FROM event_outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.OutboxEvent{}
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
