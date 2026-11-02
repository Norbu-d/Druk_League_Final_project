package natsadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

type MatchFinalizer interface {
	NotifyMatchFinalized(context.Context, domain.MatchFinalized) error
}

func Handler(service MatchFinalizer) eventkit.Handler {
	return func(ctx context.Context, subject string, payload []byte) error {
		if subject != eventkit.SubjectMatchFinalized {
			return eventkit.Permanent(fmt.Errorf("unexpected subject %q", subject))
		}
		var event domain.MatchFinalized
		if err := json.Unmarshal(payload, &event); err != nil {
			return eventkit.Permanent(fmt.Errorf("decode MatchFinalized event: %w", err))
		}
		if err := event.Validate(); err != nil {
			return eventkit.Permanent(err)
		}
		if err := service.NotifyMatchFinalized(ctx, event); err != nil {
			if errors.Is(err, core.ErrInvalid) || errors.Is(err, core.ErrNotFound) || errors.Is(err, core.ErrPrecondition) {
				return eventkit.Permanent(err)
			}
			return err
		}
		return nil
	}
}

func MatchFinalizedConsumer(service MatchFinalizer) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{
		Durable:    "notification-match-finalized-v1",
		Subject:    eventkit.SubjectMatchFinalized,
		MaxDeliver: 5,
		Backoff:    2 * time.Second,
		Handler:    Handler(service),
	}
}
