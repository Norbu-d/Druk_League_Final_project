package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"github.com/drukleague/platform/services/registration/internal/ports"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,tournament_id::text,team_id::text,status,COALESCE(registered_by::text,''),COALESCE(reviewed_by::text,''),review_reason,submitted_at`

func (r *Repository) Create(ctx context.Context, reg domain.Registration) (domain.Registration, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO registrations(tournament_id,team_id,registered_by) VALUES($1,$2,NULLIF($3,'')::uuid) RETURNING `+columns, reg.TournamentID, reg.TeamID, reg.RegisteredBy))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.Registration{}, core.Conflict("team is already registered for this tournament")
		}
		return domain.Registration{}, err
	}
	return created, nil
}

// Review only changes a registration that is still pending. If nothing was updated, it
// tells "already decided the same way" (fine, repeat is a no-op) from "decided differently".
func (r *Repository) Review(ctx context.Context, id, decision, reason, reviewerID string) (domain.Registration, error) {
	reg, err := scan(r.db.QueryRowContext(ctx, `UPDATE registrations SET status=$2,review_reason=$3,reviewed_by=NULLIF($4,'')::uuid,reviewed_at=now() WHERE id=$1 AND status='pending' RETURNING `+columns, id, decision, reason, reviewerID))
	if err == nil {
		return reg, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Registration{}, err
	}
	reg, err = scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM registrations WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Registration{}, core.NotFound("registration not found")
	}
	if err != nil {
		return domain.Registration{}, err
	}
	if reg.Status != decision {
		return domain.Registration{}, core.Precondition("registration has already been reviewed")
	}
	return reg, nil
}

func (r *Repository) List(ctx context.Context, f ports.Filter, limit, offset int) ([]domain.Registration, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM registrations WHERE ($1='' OR tournament_id::text=$1) AND ($2='' OR status=$2) AND ($3='' OR registered_by::text=$3) ORDER BY submitted_at DESC,id LIMIT $4 OFFSET $5`, f.TournamentID, f.Status, f.RegisteredBy, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Registration, 0, limit)
	for rows.Next() {
		reg, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return items, more, nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Registration, error) {
	var reg domain.Registration
	err := row.Scan(&reg.ID, &reg.TournamentID, &reg.TeamID, &reg.Status, &reg.RegisteredBy, &reg.ReviewedBy, &reg.ReviewReason, &reg.SubmittedAt)
	return reg, err
}
