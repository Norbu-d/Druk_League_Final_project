package natsadapter

import (
	"context"
	"log"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/fixture/internal/ports"
)

// RelayOnce publishes pending FixturesGenerated events in order, stopping at the first failure.
func RelayOnce(ctx context.Context, outbox ports.Outbox, js eventkit.Publisher) (int, error) {
	events, err := outbox.PendingEvents(ctx, 50)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		if err := eventkit.Publish(js, eventkit.SubjectFixturesGenerated, event.Payload, "fixtures-generated-"+event.EventID); err != nil {
			return published, err
		}
		if err := outbox.MarkPublished(ctx, event.EventID); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

// RunRelay polls the outbox until ctx is cancelled.
func RunRelay(ctx context.Context, outbox ports.Outbox, js eventkit.Publisher, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := RelayOnce(ctx, outbox, js); err != nil {
				log.Printf("fixture outbox relay: %v", err)
			}
		}
	}
}
