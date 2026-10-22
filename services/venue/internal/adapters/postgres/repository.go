package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/venue/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,name,dzongkhag,capacity,created_at`

func (r *Repository) Create(ctx context.Context, v domain.Venue) (domain.Venue, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO venues(name,dzongkhag,capacity) VALUES($1,$2,$3) RETURNING `+columns, v.Name, v.Dzongkhag, v.Capacity))
	if err != nil && strings.Contains(err.Error(), "venues_name_dzongkhag_unique") {
		return domain.Venue{}, core.Conflict("a venue with this name already exists in the dzongkhag")
	}
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Venue, error) {
	v, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM venues WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Venue{}, core.NotFound("venue not found")
	}
	return v, err
}

func (r *Repository) List(ctx context.Context, dzongkhag string, limit, offset int) ([]domain.Venue, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM venues WHERE ($1='' OR dzongkhag=$1) ORDER BY dzongkhag,name,id LIMIT $2 OFFSET $3`, dzongkhag, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Venue, 0, limit)
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		if len(items) == limit {
			return items, true, nil
		}
		items = append(items, v)
	}
	return items, false, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Venue, error) {
	var v domain.Venue
	err := row.Scan(&v.ID, &v.Name, &v.Dzongkhag, &v.Capacity, &v.CreatedAt)
	return v, err
}
