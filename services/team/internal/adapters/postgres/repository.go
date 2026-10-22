package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/team/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,name,short_code,dzongkhag,COALESCE(tournament_id::text,''),COALESCE(manager_id::text,''),created_at`

func (r *Repository) Create(ctx context.Context, t domain.Team) (domain.Team, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO teams(name,short_code,dzongkhag,tournament_id,manager_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid) RETURNING `+columns, t.Name, t.ShortCode, t.Dzongkhag, t.TournamentID, t.ManagerID))
	if err != nil && strings.Contains(err.Error(), "teams_tournament_code_unique") {
		return domain.Team{}, core.Conflict("team short code already exists for this tournament")
	}
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Team, error) {
	t, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM teams WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Team{}, core.NotFound("team not found")
	}
	return t, err
}

func (r *Repository) List(ctx context.Context, f domain.Filter, limit, offset int) ([]domain.Team, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM teams WHERE ($1='' OR tournament_id::text=$1) AND ($2='' OR dzongkhag=$2) AND ($3='' OR manager_id::text=$3) ORDER BY name,id LIMIT $4 OFFSET $5`, f.TournamentID, f.Dzongkhag, f.ManagerID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Team, 0, limit)
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

func scan(row scanner) (domain.Team, error) {
	var t domain.Team
	err := row.Scan(&t.ID, &t.Name, &t.ShortCode, &t.Dzongkhag, &t.TournamentID, &t.ManagerID, &t.CreatedAt)
	return t, err
}
