package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/tournament/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,name,sport,season,status,venue,starts_at,dzongkhag`

func (r *Repository) Create(ctx context.Context, t domain.Tournament) (domain.Tournament, error) {
	var starts any
	if t.StartsAt != nil {
		starts = *t.StartsAt
	}
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO tournaments(name,sport,season,status,venue,starts_at,dzongkhag) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+columns, t.Name, t.Sport, t.Season, t.Status, t.Venue, starts, t.Dzongkhag))
	if err != nil && strings.Contains(err.Error(), "tournaments_name_season_unique") {
		return domain.Tournament{}, core.Conflict("a tournament with this name already exists for the season")
	}
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Tournament, error) {
	t, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM tournaments WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Tournament{}, core.NotFound("tournament not found")
	}
	return t, err
}

func (r *Repository) List(ctx context.Context, season string, limit, offset int) ([]domain.Tournament, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM tournaments WHERE ($1='' OR season=$1) ORDER BY season DESC,name,id LIMIT $2 OFFSET $3`, season, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Tournament, 0, limit)
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		if len(items) == limit {
			return items, true, nil
		}
		items = append(items, t)
	}
	return items, false, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Tournament, error) {
	var t domain.Tournament
	var starts sql.NullTime
	if err := row.Scan(&t.ID, &t.Name, &t.Sport, &t.Season, &t.Status, &t.Venue, &starts, &t.Dzongkhag); err != nil {
		return domain.Tournament{}, err
	}
	if starts.Valid {
		utc := starts.Time.UTC()
		t.StartsAt = &utc
	}
	return t, nil
}
