package ports

import (
	"context"

	"github.com/drukleague/platform/services/user/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, user domain.User, passwordHash string) (domain.User, error)
	Get(ctx context.Context, id string) (domain.User, error)
	// Credentials returns the user and stored hash for a username.
	Credentials(ctx context.Context, username string) (domain.User, string, error)
	UpdatePasswordHash(ctx context.Context, id, passwordHash string) error
	List(ctx context.Context, role string, limit, offset int) ([]domain.User, bool, error)
	// CreateFirstSuperAdmin inserts the admin only when no super admin exists yet.
	CreateFirstSuperAdmin(ctx context.Context, username, passwordHash string) (bool, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	// Verify reports whether password matches and whether the hash should be upgraded.
	// An empty hash still costs one bcrypt comparison so unknown usernames are not faster.
	Verify(hash, password string) (ok bool, needsRehash bool)
}
