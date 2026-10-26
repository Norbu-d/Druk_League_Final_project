package ports

import (
	"context"

	"github.com/drukleague/platform/services/official/internal/domain"
)

type Repository interface {
	// Record stores the current assignment for a match, replacing an earlier one.
	Record(context.Context, domain.Assignment) (domain.Assignment, error)
	List(ctx context.Context, officialUserID, tournamentID string) ([]domain.Assignment, error)
}

// Matches assigns the official on the Match service, which checks the user's role and
// refuses completed matches.
type Matches interface {
	AssignOfficial(ctx context.Context, matchID, officialUserID string) (domain.MatchRef, error)
}
