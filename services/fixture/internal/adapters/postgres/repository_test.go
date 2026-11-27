package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

const (
	tournament  = "11111111-1111-4111-8111-111111111111"
	teamA       = "22222222-2222-4222-8222-222222222222"
	teamB       = "33333333-3333-4333-8333-333333333333"
	teamC       = "44444444-4444-4444-8444-444444444444"
	venue       = "55555555-5555-4555-8555-555555555555"
	eventID     = "66666666-6666-4666-8666-666666666666"
	matchID     = "77777777-7777-4777-8777-777777777777"
	tournament2 = "88888888-8888-4888-8888-888888888888"
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
	if err := repo.LinkMatch(ctx, stored[0].ID, matchID); err != nil {
		t.Fatal(err)
	}
	page, more, err := repo.List(ctx, tournament, 2, 0)
	if err != nil || len(page) != 2 || !more || page[0].MatchID != matchID || !page[0].Kickoff.Equal(start) {
		t.Fatalf("list: %v %v %+v", err, more, page)
	}
	lastPage, more, err := repo.List(ctx, tournament, 2, 2)
	if err != nil || more || len(lastPage) != 1 || lastPage[0].ID != stored[2].ID {
		t.Fatalf("last page: %v %v %+v", err, more, lastPage)
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
	if items, more, err := repo.List(ctx, tournament, 10, 0); err != nil || more || len(items) != 3 {
		t.Fatalf("complete list: %v %v %+v", err, more, items)
	}
	if items, more, err := repo.List(ctx, "88888888-8888-4888-8888-888888888888", 10, 0); err != nil || more || len(items) != 0 {
		t.Fatalf("empty list: %v %v %+v", err, more, items)
	}
	if _, _, err := repo.SaveSchedule(ctx, "not-a-uuid", nil, domain.OutboxEvent{}); err == nil {
		t.Fatal("invalid tournament id accepted")
	}
	if _, _, err := repo.SaveSchedule(ctx, "88888888-8888-4888-8888-888888888888", []domain.Fixture{{HomeTeamID: "not-a-uuid"}}, domain.OutboxEvent{}); err == nil {
		t.Fatal("invalid team id accepted")
	}
	if _, _, err := repo.SaveSchedule(ctx, "99999999-9999-4999-8999-999999999999", nil, domain.OutboxEvent{EventID: "not-a-uuid", Payload: []byte(`{}`)}); err == nil {
		t.Fatal("invalid event id accepted")
	}
	if _, _, err := repo.List(ctx, "not-a-uuid", 10, 0); err == nil {
		t.Fatal("invalid list tournament id accepted")
	}
	if err := repo.LinkMatch(ctx, "not-a-uuid", "not-a-uuid"); err == nil {
		t.Fatal("invalid match link ids accepted")
	}
	if _, err := repo.PendingEvents(ctx, -1); err == nil {
		t.Fatal("negative pending-event limit accepted")
	}
	if err := repo.MarkPublished(ctx, "not-a-uuid"); err == nil {
		t.Fatal("invalid event id accepted when publishing")
	}
}

func TestSaveScheduleRollsBackWhenOutboxInsertFails(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	schedule, err := domain.Schedule(tournament, []string{teamA, teamB}, nil, time.Now(), domain.FormatRoundRobin)
	if err != nil {
		t.Fatal(err)
	}
	event := domain.OutboxEvent{EventID: eventID, Payload: []byte(`{}`)}
	if _, created, err := repo.SaveSchedule(ctx, tournament, schedule, event); err != nil || !created {
		t.Fatalf("initial save: created=%v err=%v", created, err)
	}
	if _, _, err := repo.SaveSchedule(ctx, tournament2, schedule, event); err == nil {
		t.Fatal("duplicate outbox event was accepted")
	}
	items, more, err := repo.List(ctx, tournament2, 10, 0)
	if err != nil || more || len(items) != 0 {
		t.Fatalf("failed transaction left fixtures: items=%+v more=%v err=%v", items, more, err)
	}
}

func TestRepositoryReturnsClosedDatabaseErrors(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost/postgres")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	ctx := context.Background()
	fixture := domain.Fixture{TournamentID: tournament, HomeTeamID: teamA, AwayTeamID: teamB}
	event := domain.OutboxEvent{EventID: eventID, Payload: []byte(`{}`)}
	if _, _, err := repo.SaveSchedule(ctx, tournament, []domain.Fixture{fixture}, event); err == nil {
		t.Fatal("save with closed database succeeded")
	}
	if err := repo.LinkMatch(ctx, "fixture", matchID); err == nil {
		t.Fatal("link with closed database succeeded")
	}
	if _, _, err := repo.List(ctx, tournament, 10, 0); err == nil {
		t.Fatal("list with closed database succeeded")
	}
	if _, err := repo.PendingEvents(ctx, 10); err == nil {
		t.Fatal("outbox query with closed database succeeded")
	}
	if err := repo.MarkPublished(ctx, eventID); err == nil {
		t.Fatal("mark with closed database succeeded")
	}
}
