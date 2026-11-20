package natsadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/search/internal/domain"
)

type captureWriter struct {
	event domain.IndexEvent
	err   error
	calls int
}

func (w *captureWriter) ApplyIndexEvent(_ context.Context, event domain.IndexEvent) error {
	w.calls++
	w.event = event
	return w.err
}

const validTeamEvent = `{"event_id":"22222222-2222-4222-8222-222222222222","entity_type":"team","entity_id":"11111111-1111-4111-8111-111111111111","version":1,"operation":"upsert","occurred_at":"2026-11-20T00:00:00Z","fields":{"title":"Thimphu FC"}}`

func TestHandlerAppliesValidEvent(t *testing.T) {
	writer := &captureWriter{}
	if err := Handler(writer, domain.EntityTeam)(context.Background(), eventkit.SubjectSearchTeam, []byte(validTeamEvent)); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || writer.event.EntityID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("writer = %+v", writer)
	}
}

func TestHandlerMarksMalformedEventPermanent(t *testing.T) {
	writer := &captureWriter{}
	err := Handler(writer, domain.EntityTeam)(context.Background(), eventkit.SubjectSearchTeam, []byte(`{`))
	if !eventkit.IsPermanent(err) || writer.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, writer.calls)
	}
}

func TestHandlerReturnsStorageError(t *testing.T) {
	want := errors.New("database unavailable")
	err := Handler(&captureWriter{err: want}, domain.EntityTeam)(context.Background(), eventkit.SubjectSearchTeam, []byte(validTeamEvent))
	if !errors.Is(err, want) || eventkit.IsPermanent(err) {
		t.Fatalf("error = %v", err)
	}
}

func TestConsumersConfigureEachSearchEntity(t *testing.T) {
	consumers := Consumers(&captureWriter{})
	if len(consumers) != 3 {
		t.Fatalf("got %d consumers", len(consumers))
	}
	wantSubjects := []string{eventkit.SubjectSearchTeam, eventkit.SubjectSearchPlayer, eventkit.SubjectSearchTournament}
	for i, subject := range wantSubjects {
		if consumers[i].Subject != subject || consumers[i].Durable == "" || consumers[i].Handler == nil {
			t.Fatalf("consumer %d = %+v", i, consumers[i])
		}
	}
}
