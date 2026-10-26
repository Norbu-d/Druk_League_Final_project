package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/official/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,match_id::text,official_user_id::text,COALESCE(tournament_id::text,''),COALESCE(assigned_by::text,''),assigned_at`

func (r *Repository) Record(ctx context.Context, a domain.Assignment) (domain.Assignment, error) {
	return scan(r.db.QueryRowContext(ctx, `INSERT INTO official_assignments(match_id,official_user_id,tournament_id,assigned_by) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid)
		ON CONFLICT(match_id) DO UPDATE SET official_user_id=EXCLUDED.official_user_id,tournament_id=EXCLUDED.tournament_id,assigned_by=EXCLUDED.assigned_by,assigned_at=now()
		RETURNING `+columns, a.MatchID, a.OfficialUserID, a.TournamentID, a.AssignedBy))
}

func (r *Repository) List(ctx context.Context, officialUserID, tournamentID string) ([]domain.Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM official_assignments WHERE ($1='' OR official_user_id::text=$1) AND ($2='' OR tournament_id::text=$2) ORDER BY assigned_at DESC,id LIMIT 500`, officialUserID, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Assignment{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Assignment, error) {
	var a domain.Assignment
	err := row.Scan(&a.ID, &a.MatchID, &a.OfficialUserID, &a.TournamentID, &a.AssignedBy, &a.AssignedAt)
	return a, err
}
