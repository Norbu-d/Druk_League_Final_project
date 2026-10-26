package ports

import (
	"context"

	"github.com/drukleague/platform/services/player/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Player) (domain.Player, error)
	Get(context.Context, string) (domain.Player, error)
	List(ctx context.Context, teamID string, limit, offset int) ([]domain.Player, bool, error)
	Update(context.Context, domain.Player) (domain.Player, error)
}

// Teams reads the team a player belongs to, to confirm it exists and who manages it.
type Teams interface {
	Get(context.Context, string) (domain.TeamRef, error)
}
