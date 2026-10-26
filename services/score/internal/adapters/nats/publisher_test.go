package natsadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/services/score/internal/domain"
	"github.com/nats-io/nats.go"
)

type fakeOutbox struct {
	pending   []domain.OutboxEvent
	published []string
	readErr   error
	markErr   error
}

func (f *fakeOutbox) PendingEvents(context.Context, int) ([]domain.OutboxEvent, error) {
	return f.pending, f.readErr
}
func (f *fakeOutbox) MarkPublished(_ context.Context, id string) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.published = append(f.published, id)
	return nil
}

type fakeJS struct {
	msgs   []*nats.Msg
	failAt int
}

func (f *fakeJS) PublishMsg(m *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	if f.failAt > 0 && len(f.msgs)+1 == f.failAt {
		return nil, errors.New("nats down")
	}
	f.msgs = append(f.msgs, m)
	return &nats.PubAck{}, nil
}

func outbox() *fakeOutbox {
	return &fakeOutbox{pending: []domain.OutboxEvent{{EventID: "e1", Payload: []byte(`{}`)}, {EventID: "e2", Payload: []byte(`{}`)}}}
}

func TestRelayPublishesInOrderAndMarks(t *testing.T) {
	store, js := outbox(), &fakeJS{}
	n, err := RelayOnce(context.Background(), store, js)
	if err != nil || n != 2 || len(store.published) != 2 || store.published[0] != "e1" {
		t.Fatalf("relay: %d %v %v", n, err, store.published)
	}
	if js.msgs[0].Subject != "drukleague.match.finalized" {
		t.Fatalf("subject %s", js.msgs[0].Subject)
	}
}

func TestRelayStopsAtFirstFailureSoEventsStayQueued(t *testing.T) {
	store, js := outbox(), &fakeJS{failAt: 2}
	n, err := RelayOnce(context.Background(), store, js)
	if err == nil || n != 1 || len(store.published) != 1 {
		t.Fatalf("expected one published then stop: %d %v %v", n, err, store.published)
	}
	if _, err := RelayOnce(context.Background(), &fakeOutbox{readErr: errors.New("db")}, js); err == nil {
		t.Fatal("read error ignored")
	}
	if _, err := RelayOnce(context.Background(), &fakeOutbox{pending: outbox().pending, markErr: errors.New("db")}, &fakeJS{}); err == nil {
		t.Fatal("mark error ignored")
	}
}

func TestRunRelayStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := outbox()
	done := make(chan struct{})
	go func() { RunRelay(ctx, store, &fakeJS{}, 5*time.Millisecond); close(done) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("relay did not stop")
	}
	if len(store.published) == 0 {
		t.Fatal("relay never ran")
	}
	failing := &fakeOutbox{readErr: errors.New("db")}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	RunRelay(ctx2, failing, &fakeJS{}, 5*time.Millisecond)
}
