package ports

import (
	"context"

	"github.com/drukleague/platform/services/registration/internal/domain"
)

// Filter narrows a registration listing. Empty fields match everything.
type Filter struct{ TournamentID, Status, RegisteredBy string }

type Repository interface {
	// Create stores a pending registration; it returns core.ErrConflict when the team
	// is already registered for the tournament.
	Create(context.Context, domain.Registration) (domain.Registration, error)
	// Review records the decision on a pending registration. Repeating the same decision
	// is a no-op; a different decision returns core.ErrPrecondition, an unknown id core.ErrNotFound.
	Review(ctx context.Context, id, decision, reason, reviewerID string) (domain.Registration, error)
	List(ctx context.Context, filter Filter, limit, offset int) ([]domain.Registration, bool, error)
}

// Teams looks a team up in the Team service.
type Teams interface {
	Get(ctx context.Context, id string) (domain.TeamRef, error)
}

// Tournaments looks a tournament up in the Tournament service.
type Tournaments interface {
	Get(ctx context.Context, id string) (domain.TournamentRef, error)
}
