package natsadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/core"
)

type matchFinalized struct {
	EventID      string    `json:"event_id"`
	MatchID      string    `json:"match_id"`
	TournamentID string    `json:"tournament_id"`
	HomeTeamID   string    `json:"home_team_id"`
	AwayTeamID   string    `json:"away_team_id"`
	HomeScore    int32     `json:"home_score"`
	AwayScore    int32     `json:"away_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
}

// FinalizedMatchHandler validates and observes MatchFinalized events. Fixture has no
// post-finalization state to mutate today, so acknowledging a valid event is idempotent.
// Permanent payload errors go straight to eventkit's DLQ; transient handler errors can
// be retried by the shared consumer.
func FinalizedMatchHandler() eventkit.Handler {
	return func(_ context.Context, subject string, payload []byte) error {
		if subject != eventkit.SubjectMatchFinalized {
			return eventkit.Permanent(fmt.Errorf("unexpected subject %q", subject))
		}
		var event matchFinalized
		if err := json.Unmarshal(payload, &event); err != nil {
			return eventkit.Permanent(fmt.Errorf("decode MatchFinalized event: %w", err))
		}
		if !core.IsUUID(event.EventID) || !core.IsUUID(event.MatchID) || !core.IsUUID(event.TournamentID) ||
			!core.IsUUID(event.HomeTeamID) || !core.IsUUID(event.AwayTeamID) || event.HomeTeamID == event.AwayTeamID ||
			event.HomeScore < 0 || event.HomeScore > 99 || event.AwayScore < 0 || event.AwayScore > 99 || event.FinalizedAt.IsZero() {
			return eventkit.Permanent(errInvalidMatchFinalized)
		}
		log.Printf("fixture observed finalized match %s for tournament %s (event %s)", event.MatchID, event.TournamentID, event.EventID)
		return nil
	}
}

var errInvalidMatchFinalized = errors.New("invalid MatchFinalized event")

// FinalizedMatchConsumer uses eventkit's explicit ack, five-attempt redelivery and DLQ handling.
func FinalizedMatchConsumer() eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{
		Durable:    "fixture-match-finalized-v1",
		Subject:    eventkit.SubjectMatchFinalized,
		MaxDeliver: 5,
		Backoff:    2 * time.Second,
		Handler:    FinalizedMatchHandler(),
	}
}
