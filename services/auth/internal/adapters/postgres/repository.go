package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/auth/internal/domain"
)

// Repository implements both the Sessions and Attempts ports.
type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const sessionColumns = `id::text,user_id::text,expires_at,revoked_at IS NOT NULL`

func (r *Repository) Create(ctx context.Context, userID, refreshHash string, expires time.Time) (domain.Session, error) {
	return scanSession(r.db.QueryRowContext(ctx, `INSERT INTO auth_sessions(user_id,token_hash,expires_at) VALUES($1,$2,$3) RETURNING `+sessionColumns, userID, refreshHash, expires))
}

func (r *Repository) ByRefreshHash(ctx context.Context, refreshHash string) (domain.Session, error) {
	session, err := scanSession(r.db.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM auth_sessions WHERE token_hash=$1`, refreshHash))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, core.NotFound("session not found")
	}
	return session, err
}

func (r *Repository) Rotate(ctx context.Context, sessionID, oldHash, newHash string, expires time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE auth_sessions SET token_hash=$3,expires_at=$4,last_used_at=now() WHERE id=$1 AND token_hash=$2 AND revoked_at IS NULL`, sessionID, oldHash, newHash, expires)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Session, error) {
	session, err := scanSession(r.db.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM auth_sessions WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, core.NotFound("session not found")
	}
	return session, err
}

func (r *Repository) Revoke(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1`, id)
	return err
}

func scanSession(row interface{ Scan(...any) error }) (domain.Session, error) {
	var s domain.Session
	err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.Revoked)
	return s, err
}

// Attempts is the login_attempts adapter.
type Attempts struct{ db *sql.DB }

func NewAttempts(db *sql.DB) *Attempts { return &Attempts{db: db} }

func (a *Attempts) Get(ctx context.Context, username string) (domain.Attempts, error) {
	var result domain.Attempts
	var window, locked sql.NullTime
	err := a.db.QueryRowContext(ctx, `SELECT failed,window_start,locked_until FROM login_attempts WHERE username=$1`, username).Scan(&result.Failed, &window, &locked)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Attempts{}, nil
	}
	result.WindowStart, result.LockedUntil = window.Time, locked.Time
	return result, err
}

func (a *Attempts) Save(ctx context.Context, username string, attempts domain.Attempts) error {
	_, err := a.db.ExecContext(ctx, `INSERT INTO login_attempts(username,failed,window_start,locked_until) VALUES($1,$2,$3,$4)
		ON CONFLICT(username) DO UPDATE SET failed=EXCLUDED.failed,window_start=EXCLUDED.window_start,locked_until=EXCLUDED.locked_until`,
		username, attempts.Failed, nullTime(attempts.WindowStart), nullTime(attempts.LockedUntil))
	return err
}

func (a *Attempts) Reset(ctx context.Context, username string) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM login_attempts WHERE username=$1`, username)
	return err
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
