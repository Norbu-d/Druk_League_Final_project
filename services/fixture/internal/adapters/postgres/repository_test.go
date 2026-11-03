package postgres

import (
	"context"
	"fmt"
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
	if err := repo.LinkMatch(ctx, stored[0].ID, matchID); err != nil {
		t.Fatal(err)
	}
	page, more, err := repo.List(ctx, tournament, 2, 0)
	if err != nil || len(page) != 2 || !more || page[0].MatchID != matchID || !page[0].Kickoff.Equal(start) {
		t.Fatalf("list: %v %v %+v", err, more, page)
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

func TestKnockoutFinalizedEventsCreateOneNextRoundMatch(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	knockoutTournament := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	ranked := []string{
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
		"33333333-3333-4333-8333-333333333333",
		"44444444-4444-4444-8444-444444444444",
	}
	plan, err := domain.KnockoutSchedule(knockoutTournament, ranked, nil, time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC))
	if err != nil || len(plan.Fixtures) != 2 {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	if _, created, err := repo.SaveKnockoutSchedule(ctx, knockoutTournament, plan, domain.OutboxEvent{EventID: "55555555-5555-4555-8555-555555555555", Payload: []byte(`{"type":"FixturesGenerated"}`)}); err != nil || !created {
		t.Fatalf("save bracket: created=%v err=%v", created, err)
	}
	fixtures, _, err := repo.List(ctx, knockoutTournament, 10, 0)
	if err != nil || len(fixtures) != 2 {
		t.Fatalf("list bracket: %d %v", len(fixtures), err)
	}
	matchOne := "66666666-6666-4666-8666-666666666666"
	matchTwo := "77777777-7777-4777-8777-777777777777"
	if err := repo.LinkMatch(ctx, fixtures[0].ID, matchOne); err != nil {
		t.Fatal(err)
	}
	if err := repo.LinkMatch(ctx, fixtures[1].ID, matchTwo); err != nil {
		t.Fatal(err)
	}
	first := finalizedEvent(knockoutTournament, "88888888-8888-4888-8888-888888888888", matchOne, fixtures[0], 2, 1)
	if _, created, err := repo.AdvanceKnockout(ctx, first); err != nil || created {
		t.Fatalf("advanced before both feeders finished: created=%v err=%v", created, err)
	}
	second := finalizedEvent(knockoutTournament, "99999999-9999-4999-8999-999999999999", matchTwo, fixtures[1], 1, 1)
	next, created, err := repo.AdvanceKnockout(ctx, second)
	if err != nil || !created || next.Round != 2 {
		t.Fatalf("next round: %+v created=%v err=%v", next, created, err)
	}
	if next.HomeTeamID != fixtures[0].HomeTeamID || next.AwayTeamID != fixtures[1].HomeTeamID {
		t.Fatalf("next fixture has wrong winners: %+v", next)
	}
	pending, retryCreated, err := repo.AdvanceKnockout(ctx, second)
	if err != nil || !retryCreated || pending.ID != next.ID {
		t.Fatalf("duplicate event did not return pending match for retry: %+v created=%v err=%v", pending, retryCreated, err)
	}
	if err := repo.LinkMatch(ctx, next.ID, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"); err != nil {
		t.Fatal(err)
	}
	if _, created, err := repo.AdvanceKnockout(ctx, second); err != nil || created {
		t.Fatalf("linked duplicate event returned a new match: created=%v err=%v", created, err)
	}
}

func TestSaveKnockoutScheduleAppendsAfterLeagueFixtures(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	knockoutTournament := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	ranked := []string{
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
		"33333333-3333-4333-8333-333333333333",
		"44444444-4444-4444-8444-444444444444",
	}
	league, err := domain.Schedule(knockoutTournament, ranked, nil, time.Date(2026, 10, 1, 4, 0, 0, 0, time.UTC), domain.FormatLeagueKnockout)
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := repo.SaveSchedule(ctx, knockoutTournament, league, domain.OutboxEvent{EventID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Payload: []byte(`{"stage":"league"}`)}); err != nil || !created {
		t.Fatalf("save league stage: created=%v err=%v", created, err)
	}
	plan, err := domain.KnockoutSchedule(knockoutTournament, ranked, nil, time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	stored, created, err := repo.SaveKnockoutSchedule(ctx, knockoutTournament, plan, domain.OutboxEvent{EventID: "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", Payload: []byte(`{"stage":"knockout"}`)})
	if err != nil || !created || len(stored) != 2 {
		t.Fatalf("save knockout stage: fixtures=%d created=%v err=%v", len(stored), created, err)
	}
	for _, fixture := range stored {
		if fixture.Format != domain.FormatKnockout || fixture.Round != 1 {
			t.Fatalf("unexpected knockout fixture: %+v", fixture)
		}
	}
}

func TestLeagueKnockoutWaitsForEveryFinalizedFixture(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	tournamentID := "ffffffff-ffff-4fff-8fff-ffffffffffff"
	teams := []string{teamA, teamB, teamC, "88888888-8888-4888-8888-888888888888"}
	group, err := domain.Schedule(tournamentID, teams, nil, time.Date(2026, 10, 1, 4, 0, 0, 0, time.UTC), domain.FormatLeagueKnockout)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.SaveSchedule(ctx, tournamentID, group, domain.OutboxEvent{EventID: "99999999-9999-4999-8999-999999999999", Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	stored, _, err := repo.List(ctx, tournamentID, 20, 0)
	if err != nil || len(stored) != len(group) {
		t.Fatalf("group fixtures: %d %v", len(stored), err)
	}
	for i := range stored {
		matchID := fmt.Sprintf("%08x-aaaa-4aaa-8aaa-%012x", i+1, i+1)
		eventID := fmt.Sprintf("%08x-bbbb-4bbb-8bbb-%012x", i+1, i+1)
		if err := repo.LinkMatch(ctx, stored[i].ID, matchID); err != nil {
			t.Fatal(err)
		}
		event := finalizedEvent(tournamentID, eventID, matchID, stored[i], 1, 0)
		_, created, err := repo.AdvanceKnockout(ctx, event)
		if err != nil {
			t.Fatal(err)
		}
		if created != (i == len(stored)-1) {
			t.Fatalf("fixture %d: created=%v, want only last fixture to start knockout", i, created)
		}
	}
}

func finalizedEvent(tournamentID, eventID, matchID string, fixture domain.Fixture, home, away int32) domain.MatchFinalized {
	return domain.MatchFinalized{EventID: eventID, MatchID: matchID, TournamentID: tournamentID, HomeTeamID: fixture.HomeTeamID, AwayTeamID: fixture.AwayTeamID,
		HomeScore: home, AwayScore: away, FinalizedAt: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
}
