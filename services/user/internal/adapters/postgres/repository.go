package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/user/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,username,role,created_at`

func (r *Repository) Create(ctx context.Context, u domain.User, passwordHash string) (domain.User, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO users(username,password_hash,role) VALUES($1,$2,$3) RETURNING `+columns, u.Username, passwordHash, u.Role))
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.User{}, core.Conflict("username already exists")
	}
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.User, error) {
	u, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, core.NotFound("user not found")
	}
	return u, err
}

func (r *Repository) Credentials(ctx context.Context, username string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := r.db.QueryRowContext(ctx, `SELECT `+columns+`,password_hash FROM users WHERE lower(username)=lower($1)`, username).Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", core.NotFound("user not found")
	}
	return u, hash, err
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash=$2,password_changed_at=now() WHERE id=$1`, id, passwordHash)
	return err
}

func (r *Repository) List(ctx context.Context, role string, limit, offset int) ([]domain.User, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM users WHERE ($1='' OR role=$1) ORDER BY username,id LIMIT $2 OFFSET $3`, role, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.User, 0, limit)
	for rows.Next() {
		u, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		if len(items) == limit {
			return items, true, nil
		}
		items = append(items, u)
	}
	return items, false, rows.Err()
}

// CreateFirstSuperAdmin is safe to run on every start: a transaction-scoped advisory lock
// stops two replicas racing, and nothing happens once any super admin exists.
func (r *Repository) CreateFirstSuperAdmin(ctx context.Context, username, passwordHash string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('user-bootstrap'))`); err != nil {
		return false, err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role='super_admin')`).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO users(username,password_hash,role) VALUES($1,$2,'super_admin') ON CONFLICT(username) DO NOTHING`, username, passwordHash); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
	return u, err
}
