package ports

import (
	"context"
	"time"

	"github.com/drukleague/platform/services/match/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Match) (domain.Match, error)
	Get(context.Context, string) (domain.Match, error)
	List(context.Context, string, *time.Time, int, int) ([]domain.Match, bool, error)
	AssignOfficial(context.Context, string, string) (domain.Match, error)
	// MarkCompleted applies a MatchFinalized event once; it returns false for an already-processed event.
	MarkCompleted(ctx context.Context, eventID, matchID string) (bool, error)
}

type TeamDirectory interface {
	TournamentForTeam(context.Context, string) (string, error)
}

type UserDirectory interface {
	RoleForUser(context.Context, string) (string, error)
}
