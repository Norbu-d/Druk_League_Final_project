package natsadapter

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

type EventHandler interface {
	NotifyMatchFinalized(context.Context, domain.FinalizedMatch) error
}

func Handler(service EventHandler) eventkit.Handler {
	return func(ctx context.Context, _ string, payload []byte) error {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(payload, &raw); err != nil {
			return eventkit.Permanent(err)
		}

		var event domain.FinalizedMatch
		for key, destination := range map[string]*string{
			"event_id": &event.EventID, "match_id": &event.MatchID, "home_team_id": &event.HomeTeamID, "away_team_id": &event.AwayTeamID,
		} {
			if err := json.Unmarshal(raw[key], destination); err != nil {
				return eventkit.Permanent(err)
			}
		}
		if err := json.Unmarshal(raw["home_score"], &event.HomeScore); err != nil {
			return eventkit.Permanent(err)
		}
		if err := json.Unmarshal(raw["away_score"], &event.AwayScore); err != nil {
			return eventkit.Permanent(err)
		}

		err := service.NotifyMatchFinalized(ctx, event)
		if errors.Is(err, core.ErrInvalid) {
			return eventkit.Permanent(err)
		}
		return err
	}
}

func Consumer(service EventHandler) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{
		Durable: "notification-match-finalized-v1",
		Subject: eventkit.SubjectMatchFinalized,
		Handler: Handler(service),
	}
}
