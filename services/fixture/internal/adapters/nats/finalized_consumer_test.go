package natsadapter

import (
	"context"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

func TestFinalizedMatchConsumerConfig(t *testing.T) {
	config := FinalizedMatchConsumer(nil)
	if config.Durable != "fixture-match-finalized-v1" || config.Subject != eventkit.SubjectMatchFinalized || config.MaxDeliver != 5 || config.Backoff <= 0 || config.Handler == nil {
		t.Fatalf("unexpected consumer config: %+v", config)
	}
}

func TestFinalizedMatchHandler(t *testing.T) {
	valid := []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","match_id":"22222222-2222-4222-8222-222222222222","tournament_id":"33333333-3333-4333-8333-333333333333","home_team_id":"44444444-4444-4444-8444-444444444444","away_team_id":"55555555-5555-4555-8555-555555555555","home_score":2,"away_score":1,"finalized_at":"2026-11-02T10:00:00Z"}`)
	var got domain.MatchFinalized
	handler := FinalizedMatchHandler(func(_ context.Context, event domain.MatchFinalized) error {
		got = event
		return nil
	})
	if err := handler(context.Background(), eventkit.SubjectMatchFinalized, valid); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	if got.MatchID != "22222222-2222-4222-8222-222222222222" || got.HomeScore != 2 || got.AwayScore != 1 {
		t.Fatalf("finalized event was not passed to fixture application: %+v", got)
	}

	for _, tc := range []struct {
		name    string
		subject string
		payload []byte
	}{
		{name: "wrong subject", subject: eventkit.SubjectFixturesGenerated, payload: valid},
		{name: "malformed payload", subject: eventkit.SubjectMatchFinalized, payload: []byte("{")},
		{name: "invalid event", subject: eventkit.SubjectMatchFinalized, payload: []byte(`{"event_id":"bad"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := handler(context.Background(), tc.subject, tc.payload); !eventkit.IsPermanent(err) {
				t.Fatalf("expected permanent error, got %v", err)
			}
		})
	}
}
