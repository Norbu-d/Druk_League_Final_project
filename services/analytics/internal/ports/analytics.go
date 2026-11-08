package ports

import (
	"context"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type Repository interface {
	RecordFinalized(context.Context, domain.MatchFinalized, domain.Tournament) error
	DashboardMetrics(context.Context, string, string) (domain.DashboardMetrics, error)
}

type TournamentDirectory interface {
	GetTournament(context.Context, string) (domain.Tournament, error)
}
