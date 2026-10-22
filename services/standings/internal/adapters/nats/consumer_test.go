package natsadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/standings/internal/domain"
)

type fakeApp struct{ err error }

func (f fakeApp) ApplyFinalized(context.Context, domain.MatchFinalized) (bool, error) {
	return f.err == nil, f.err
}

func TestHandler(t *testing.T) {
	payload := []byte(`{"event_id":"e"}`)
	if err := Handler(fakeApp{})(context.Background(), "", payload); err != nil {
		t.Fatal(err)
	}
	if !eventkit.IsPermanent(Handler(fakeApp{})(context.Background(), "", []byte("nope"))) {
		t.Fatal("bad JSON must be permanent")
	}
	if !eventkit.IsPermanent(Handler(fakeApp{err: domain.ErrInvalid})(context.Background(), "", payload)) {
		t.Fatal("invalid event must be permanent")
	}
	if err := Handler(fakeApp{err: domain.ErrDependency})(context.Background(), "", payload); eventkit.IsPermanent(err) || !errors.Is(err, domain.ErrDependency) {
		t.Fatal("team outage must be retried")
	}
	if Consumer(fakeApp{}).Durable != "standings-v1" {
		t.Fatal("durable name changed; existing consumer progress would be lost")
	}
}
