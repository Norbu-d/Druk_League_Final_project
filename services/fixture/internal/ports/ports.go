package ports

import (
	"context"

	"github.com/drukleague/platform/services/fixture/internal/domain"
)

type Repository interface {
	// SaveSchedule stores the schedule unless the tournament already has one, in which case
	// the stored fixtures are returned with created=false. The event is written to the
	// outbox in the same transaction when the schedule is new.
	SaveSchedule(ctx context.Context, tournamentID string, fixtures []domain.Fixture, event domain.OutboxEvent) (stored []domain.Fixture, created bool, err error)
	LinkMatch(ctx context.Context, fixtureID, matchID string) error
	List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error)
}

type Outbox interface {
	PendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
}

type Tournaments interface {
	Get(ctx context.Context, id string) (domain.Tournament, error)
}

type Teams interface {
	IDsInTournament(ctx context.Context, tournamentID string) ([]string, error)
}

type Venues interface {
	InDzongkhag(ctx context.Context, dzongkhag string) ([]domain.Venue, error)
}

type Matches interface {
	Create(ctx context.Context, fixture domain.Fixture) (matchID string, err error)
}
