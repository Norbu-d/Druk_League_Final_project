package ports

import (
	"context"

	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type Repository interface {
	RecordFinalized(context.Context, domain.MatchFinalized, domain.Tournament) error
}

type TournamentDirectory interface {
	GetTournament(context.Context, string) (domain.Tournament, error)
}
