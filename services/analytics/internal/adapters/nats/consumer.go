package natsadapter

import (
	"context"
	"encoding/json"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type EventHandler interface {
	RecordFinalized(context.Context, domain.MatchFinalized) error
}

func Handler(app EventHandler) eventkit.Handler {
	return func(ctx context.Context, _ string, payload []byte) error {
		var event domain.MatchFinalized
		if err := json.Unmarshal(payload, &event); err != nil {
			return eventkit.Permanent(err)
		}
		if err := event.NormalizeAndValidate(); err != nil {
			return eventkit.Permanent(err)
		}
		if err := app.RecordFinalized(ctx, event); err != nil {
			if err == domain.ErrInvalid {
				return eventkit.Permanent(err)
			}
			return err
		}
		return nil
	}
}

func Consumer(app EventHandler) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{Durable: "analytics-match-finalized-v1", Subject: eventkit.SubjectMatchFinalized, Handler: Handler(app)}
}
