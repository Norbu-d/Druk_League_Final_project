package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

const (
	tournament = "11111111-1111-4111-8111-111111111111"
	teamA      = "22222222-2222-4222-8222-222222222222"
	teamB      = "33333333-3333-4333-8333-333333333333"
	teamC      = "44444444-4444-4444-8444-444444444444"
	venue      = "55555555-5555-4555-8555-555555555555"
	eventID    = "66666666-6666-4666-8666-666666666666"
	matchID    = "77777777-7777-4777-8777-777777777777"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	start := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	schedule, err := domain.Schedule(tournament, []string{teamA, teamB, teamC}, []domain.Venue{{ID: venue, Name: "Changlimithang"}}, start, domain.FormatRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	stored, created, err := repo.SaveSchedule(ctx, tournament, schedule, domain.OutboxEvent{EventID: eventID, Payload: []byte(`{"fixtures":3}`)})
	if err != nil || !created || len(stored) != 3 || stored[0].VenueID != venue || stored[0].VenueName != "Changlimithang" || stored[0].ID == "" {
		t.Fatalf("save: %v %v %+v", err, created, stored)
	}
	// Second generation returns the stored schedule and does not add an event.
	again, created, err := repo.SaveSchedule(ctx, tournament, schedule, domain.OutboxEvent{EventID: matchID, Payload: []byte(`{}`)})
	if err != nil || created || len(again) != 3 || again[0].ID != stored[0].ID {
		t.Fatalf("second save: %v %v %d", err, created, len(again))
	}
	booked := stored[0]
	booked.MatchID, booked.VenueID, booked.VenueName, booked.Kickoff = matchID, "", "", start.Add(3*time.Hour)
	if err := repo.LinkMatch(ctx, booked); err != nil {
		t.Fatal(err)
	}
	page, more, err := repo.List(ctx, tournament, 2, 0)
	if err != nil || len(page) != 2 || !more || page[0].MatchID != matchID || !page[0].Kickoff.Equal(start.Add(3*time.Hour)) || page[0].VenueID != "" {
		t.Fatalf("linking should store the slot actually booked: %v %v %+v", err, more, page)
	}
	booked.MatchID = "11111111-1111-4111-8111-111111111111"
	if err := repo.LinkMatch(ctx, booked); err != nil {
		t.Fatal(err)
	}
	if again, _, _ := repo.List(ctx, tournament, 1, 0); again[0].MatchID != matchID {
		t.Fatal("a linked fixture must not be relinked")
	}
	pending, err := repo.PendingEvents(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].EventID != eventID {
		t.Fatalf("outbox: %+v %v", pending, err)
	}
	if err := repo.MarkPublished(ctx, eventID); err != nil {
		t.Fatal(err)
	}
	if pending, _ := repo.PendingEvents(ctx, 10); len(pending) != 0 {
		t.Fatal("published event still pending")
	}
}
