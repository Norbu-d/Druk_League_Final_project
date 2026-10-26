package ports

import (
	"context"

	"github.com/drukleague/platform/services/venue/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Venue) (domain.Venue, error)
	Get(context.Context, string) (domain.Venue, error)
	List(ctx context.Context, dzongkhag string, limit, offset int) ([]domain.Venue, bool, error)
}

// Dzongkhags resolves a district name or code to its canonical spelling.
type Dzongkhags interface {
	Canonical(context.Context, string) (string, error)
}
