package ports

import (
	"context"

	"github.com/drukleague/platform/services/score/internal/domain"
)

type Repository interface {
	UpdateLive(context.Context, string, int32, int32) (domain.Score, error)
	// Finalize locks the score and writes the MatchFinalized event in the same transaction.
	Finalize(ctx context.Context, match domain.Match, eventID, finalizedBy, reason string) (domain.Score, error)
	Get(context.Context, string) (domain.Score, error)
}

// Outbox is read by the NATS relay; rows are marked published only after JetStream acknowledges them.
type Outbox interface {
	PendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
}

type MatchReader interface {
	GetMatch(context.Context, string) (domain.Match, error)
}
