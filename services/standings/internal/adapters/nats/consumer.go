package natsadapter

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/standings/internal/domain"
)

type EventHandler interface {
	ApplyFinalized(context.Context, domain.MatchFinalized) (bool, error)
}

// Handler applies MatchFinalized events. Malformed or semantically invalid events are
// dead-lettered at once; Team service outages are retried with backoff.
func Handler(app EventHandler) eventkit.Handler {
	return func(ctx context.Context, _ string, payload []byte) error {
		var event domain.MatchFinalized
		if err := json.Unmarshal(payload, &event); err != nil {
			return eventkit.Permanent(err)
		}
		_, err := app.ApplyFinalized(ctx, event)
		if errors.Is(err, domain.ErrInvalid) {
			return eventkit.Permanent(err)
		}
		return err
	}
}

// Consumer keeps the durable name used since Part 2 so existing progress is preserved.
func Consumer(app EventHandler) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{Durable: "standings-v1", Subject: eventkit.SubjectMatchFinalized, Handler: Handler(app)}
}
