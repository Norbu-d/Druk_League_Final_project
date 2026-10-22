package natsadapter

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/match/internal/domain"
)

// CompletionHandler is the app-layer use case the consumer drives.
type CompletionHandler interface {
	MarkCompleted(ctx context.Context, eventID, matchID string) (bool, error)
}

type matchFinalized struct {
	EventID string `json:"event_id"`
	MatchID string `json:"match_id"`
}

// Handler turns MatchFinalized payloads into completed matches. Malformed events are
// dead-lettered at once; other failures are retried by the eventkit consumer.
func Handler(app CompletionHandler) eventkit.Handler {
	return func(ctx context.Context, _ string, payload []byte) error {
		var event matchFinalized
		if err := json.Unmarshal(payload, &event); err != nil {
			return eventkit.Permanent(err)
		}
		_, err := app.MarkCompleted(ctx, event.EventID, event.MatchID)
		if errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrNotFound) {
			return eventkit.Permanent(err)
		}
		return err
	}
}

// Consumer is the durable consumer that keeps Match status in step with final results.
func Consumer(app CompletionHandler) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{Durable: "match-status-v1", Subject: eventkit.SubjectMatchFinalized, Handler: Handler(app)}
}
