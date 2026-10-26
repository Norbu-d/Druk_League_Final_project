package ports

import (
	"context"
	"github.com/drukleague/platform/services/standings/internal/domain"
)

type Repository interface {
	TeamKnown(context.Context, string, string) (bool, error)
	ApplyFinalized(context.Context, domain.MatchFinalized, domain.Team, domain.Team) (bool, error)
	Get(context.Context, string) ([]domain.Standing, error)
}

type TeamDirectory interface {
	GetTeam(context.Context, string) (domain.Team, error)
}
