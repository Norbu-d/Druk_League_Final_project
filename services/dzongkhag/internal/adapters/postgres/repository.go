package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Get(ctx context.Context, code string) (domain.Dzongkhag, error) {
	var d domain.Dzongkhag
	err := r.db.QueryRowContext(ctx, "SELECT code,name FROM dzongkhags WHERE code=$1", code).Scan(&d.Code, &d.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return d, domain.ErrNotFound
	}
	return d, err
}
func (r *Repository) List(ctx context.Context) ([]domain.Dzongkhag, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT code,name FROM dzongkhags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Dzongkhag, 0, 20)
	for rows.Next() {
		var d domain.Dzongkhag
		if err := rows.Scan(&d.Code, &d.Name); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
