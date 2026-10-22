package ports

import (
	"context"

	"github.com/drukleague/platform/services/tournament/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Tournament) (domain.Tournament, error)
	Get(context.Context, string) (domain.Tournament, error)
	List(ctx context.Context, season string, limit, offset int) ([]domain.Tournament, bool, error)
}

// Dzongkhags resolves a district name or code to its canonical spelling.
type Dzongkhags interface {
	Canonical(context.Context, string) (string, error)
}
