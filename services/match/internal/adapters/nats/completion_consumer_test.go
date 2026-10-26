package natsadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/match/internal/domain"
)

type fakeApp struct{ err error }

func (f fakeApp) MarkCompleted(context.Context, string, string) (bool, error) {
	return f.err == nil, f.err
}

func TestHandler(t *testing.T) {
	payload := []byte(`{"event_id":"e","match_id":"m"}`)
	if err := Handler(fakeApp{})(context.Background(), eventkit.SubjectMatchFinalized, payload); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	if err := Handler(fakeApp{})(context.Background(), "", []byte("{not json")); !eventkit.IsPermanent(err) {
		t.Fatal("malformed JSON must be permanent")
	}
	for _, err := range []error{domain.ErrInvalid, domain.ErrNotFound} {
		if got := Handler(fakeApp{err: err})(context.Background(), "", payload); !eventkit.IsPermanent(got) {
			t.Fatalf("%v must be permanent", err)
		}
	}
	transient := errors.New("db down")
	if got := Handler(fakeApp{err: transient})(context.Background(), "", payload); eventkit.IsPermanent(got) || !errors.Is(got, transient) {
		t.Fatal("transient errors must be retried")
	}
	if config := Consumer(fakeApp{}); config.Durable != "match-status-v1" || config.Subject != eventkit.SubjectMatchFinalized {
		t.Fatalf("unexpected consumer %+v", config)
	}
}
