package natsadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/fixture/internal/domain"
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
	subjects []string
	err      error
}

func (f *fakeJS) PublishMsg(m *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.subjects = append(f.subjects, m.Subject)
	return &nats.PubAck{}, nil
}

func TestRelay(t *testing.T) {
	store := &fakeOutbox{pending: []domain.OutboxEvent{{EventID: "e1", Payload: []byte(`{}`)}}}
	js := &fakeJS{}
	if n, err := RelayOnce(context.Background(), store, js); err != nil || n != 1 || js.subjects[0] != eventkit.SubjectFixturesGenerated {
		t.Fatalf("relay: %d %v %v", n, err, js.subjects)
	}
	if _, err := RelayOnce(context.Background(), &fakeOutbox{readErr: errors.New("db")}, js); err == nil {
		t.Fatal("read error ignored")
	}
	if _, err := RelayOnce(context.Background(), store, &fakeJS{err: errors.New("nats")}); err == nil {
		t.Fatal("publish error ignored")
	}
	if _, err := RelayOnce(context.Background(), &fakeOutbox{pending: store.pending, markErr: errors.New("db")}, js); err == nil {
		t.Fatal("mark error ignored")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	RunRelay(ctx, &fakeOutbox{readErr: errors.New("db")}, js, 5*time.Millisecond)
}
