package ports

import (
	"context"

	"github.com/drukleague/platform/services/team/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Team) (domain.Team, error)
	Get(context.Context, string) (domain.Team, error)
	List(ctx context.Context, filter domain.Filter, limit, offset int) ([]domain.Team, bool, error)
}

// Tournaments confirms a referenced tournament exists before a team is linked to it.
type Tournaments interface {
	Get(context.Context, string) (domain.TournamentRef, error)
}

// Dzongkhags resolves a district name or code to its canonical spelling.
type Dzongkhags interface {
	Canonical(context.Context, string) (string, error)
}
