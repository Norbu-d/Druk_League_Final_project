package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/player/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,team_id::text,given_name,family_name,shirt_number,created_at`

func (r *Repository) Create(ctx context.Context, p domain.Player) (domain.Player, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO players(team_id,given_name,family_name,shirt_number) VALUES($1,$2,$3,$4) RETURNING `+columns, p.TeamID, p.GivenName, p.FamilyName, p.ShirtNumber))
	return created, shirtConflict(err)
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Player, error) {
	p, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM players WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Player{}, core.NotFound("player not found")
	}
	return p, err
}

func (r *Repository) List(ctx context.Context, teamID string, limit, offset int) ([]domain.Player, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM players WHERE ($1='' OR team_id::text=$1) ORDER BY family_name,given_name,id LIMIT $2 OFFSET $3`, teamID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Player, 0, limit)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		if len(items) == limit {
			return items, true, nil
		}
		items = append(items, p)
	}
	return items, false, rows.Err()
}

func (r *Repository) Update(ctx context.Context, p domain.Player) (domain.Player, error) {
	updated, err := scan(r.db.QueryRowContext(ctx, `UPDATE players SET given_name=$2,family_name=$3,shirt_number=$4 WHERE id=$1 RETURNING `+columns, p.ID, p.GivenName, p.FamilyName, p.ShirtNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Player{}, core.NotFound("player not found")
	}
	return updated, shirtConflict(err)
}

func shirtConflict(err error) error {
	if err != nil && strings.Contains(err.Error(), "players_team_shirt_unique") {
		return core.Conflict("another player in this team already wears that shirt number")
	}
	return err
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Player, error) {
	var p domain.Player
	err := row.Scan(&p.ID, &p.TeamID, &p.GivenName, &p.FamilyName, &p.ShirtNumber, &p.CreatedAt)
	return p, err
}
