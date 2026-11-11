package natsadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

type fakeMatchFinalizer struct {
	event domain.MatchFinalized
	err   error
	calls int
}

func (f *fakeMatchFinalizer) NotifyMatchFinalized(_ context.Context, event domain.MatchFinalized) error {
	f.calls++
	f.event = event
	return f.err
}

func TestMatchFinalizedConsumerConfig(t *testing.T) {
	config := MatchFinalizedConsumer(&fakeMatchFinalizer{})
	if config.Durable != "notification-match-finalized-v1" || config.Subject != eventkit.SubjectMatchFinalized || config.MaxDeliver != 5 || config.Backoff <= 0 || config.Handler == nil {
		t.Fatalf("unexpected consumer config: %+v", config)
	}
}

func TestMatchFinalizedHandler(t *testing.T) {
	valid := []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","match_id":"22222222-2222-4222-8222-222222222222","tournament_id":"33333333-3333-4333-8333-333333333333","home_team_id":"44444444-4444-4444-8444-444444444444","away_team_id":"55555555-5555-4555-8555-555555555555","home_score":2,"away_score":1,"finalized_at":"2026-11-30T10:00:00Z"}`)
	service := &fakeMatchFinalizer{}
	handler := Handler(service)
	if err := handler(context.Background(), eventkit.SubjectMatchFinalized, valid); err != nil || service.calls != 1 || service.event.EventID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("valid event: calls=%d event=%+v err=%v", service.calls, service.event, err)
	}
	if err := handler(context.Background(), eventkit.SubjectFixturesGenerated, valid); !eventkit.IsPermanent(err) {
		t.Fatalf("unexpected subject error = %v", err)
	}
	if err := handler(context.Background(), eventkit.SubjectMatchFinalized, []byte("{")); !eventkit.IsPermanent(err) {
		t.Fatalf("malformed event error = %v", err)
	}
	if err := handler(context.Background(), eventkit.SubjectMatchFinalized, []byte(`{"event_id":"invalid"}`)); !eventkit.IsPermanent(err) {
		t.Fatalf("invalid event error = %v", err)
	}
	transient := errors.New("database temporarily unavailable")
	if err := Handler(&fakeMatchFinalizer{err: transient})(context.Background(), eventkit.SubjectMatchFinalized, valid); !errors.Is(err, transient) || eventkit.IsPermanent(err) {
		t.Fatalf("transient error = %v", err)
	}
	permanent := errors.New("invalid recipient data")
	if err := Handler(&fakeMatchFinalizer{err: permanent})(context.Background(), eventkit.SubjectMatchFinalized, valid); err == nil {
		t.Fatal("service error was ignored")
	}
}
