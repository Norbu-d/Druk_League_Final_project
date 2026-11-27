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
	"github.com/drukleague/platform/services/fixture/internal/domain"
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
func FinalizedMatchHandler(advance func(context.Context, domain.MatchFinalized) error) eventkit.Handler {
	return func(ctx context.Context, subject string, payload []byte) error {
		if subject != eventkit.SubjectMatchFinalized {
			return eventkit.Permanent(fmt.Errorf("unexpected subject %q", subject))
		}
		var payloadEvent matchFinalized
		if err := json.Unmarshal(payload, &payloadEvent); err != nil {
			return eventkit.Permanent(fmt.Errorf("decode MatchFinalized event: %w", err))
		}
		if !core.IsUUID(payloadEvent.EventID) || !core.IsUUID(payloadEvent.MatchID) || !core.IsUUID(payloadEvent.TournamentID) ||
			!core.IsUUID(payloadEvent.HomeTeamID) || !core.IsUUID(payloadEvent.AwayTeamID) || payloadEvent.HomeTeamID == payloadEvent.AwayTeamID ||
			payloadEvent.HomeScore < 0 || payloadEvent.HomeScore > 99 || payloadEvent.AwayScore < 0 || payloadEvent.AwayScore > 99 || payloadEvent.FinalizedAt.IsZero() {
			return eventkit.Permanent(errInvalidMatchFinalized)
		}
		event := domain.MatchFinalized{EventID: payloadEvent.EventID, MatchID: payloadEvent.MatchID, TournamentID: payloadEvent.TournamentID,
			HomeTeamID: payloadEvent.HomeTeamID, AwayTeamID: payloadEvent.AwayTeamID, HomeScore: payloadEvent.HomeScore, AwayScore: payloadEvent.AwayScore, FinalizedAt: payloadEvent.FinalizedAt}
		if advance == nil {
			log.Printf("fixture observed finalized match %s for tournament %s (event %s)", event.MatchID, event.TournamentID, event.EventID)
			return nil
		}
		return advance(ctx, event)
	}
}

var errInvalidMatchFinalized = errors.New("invalid MatchFinalized event")

// FinalizedMatchConsumer uses eventkit's explicit ack, five-attempt redelivery and DLQ handling.
func FinalizedMatchConsumer(advance func(context.Context, domain.MatchFinalized) error) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{
		Durable:    "fixture-match-finalized-v1",
		Subject:    eventkit.SubjectMatchFinalized,
		MaxDeliver: 5,
		Backoff:    2 * time.Second,
		Handler:    FinalizedMatchHandler(advance),
	}
}
