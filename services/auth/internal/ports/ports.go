package ports

import (
	"context"
	"time"

	"github.com/drukleague/platform/services/auth/internal/domain"
)

// Users is the User service: it owns accounts and password hashes.
type Users interface {
	Verify(ctx context.Context, username, password string) (domain.User, error)
	Get(ctx context.Context, id string) (domain.User, error)
}

type Sessions interface {
	Create(ctx context.Context, userID, refreshHash string, expires time.Time) (domain.Session, error)
	ByRefreshHash(ctx context.Context, refreshHash string) (domain.Session, error)
	// Rotate swaps the refresh hash only if it still equals oldHash, so a refresh token
	// can be used once even when two requests race.
	Rotate(ctx context.Context, sessionID, oldHash, newHash string, expires time.Time) (bool, error)
	Get(ctx context.Context, id string) (domain.Session, error)
	Revoke(ctx context.Context, id string) error
}

type Attempts interface {
	Get(ctx context.Context, username string) (domain.Attempts, error)
	Save(ctx context.Context, username string, attempts domain.Attempts) error
	Reset(ctx context.Context, username string) error
}

// Signer issues access tokens.
type Signer interface {
	Sign(user domain.User, sessionID string, ttl time.Duration) (string, time.Time, error)
}
