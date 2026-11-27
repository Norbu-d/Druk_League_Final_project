package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/match/internal/domain"
)

const (
	tournament = "11111111-1111-4111-8111-111111111111"
	home       = "22222222-2222-4222-8222-222222222222"
	away       = "33333333-3333-4333-8333-333333333333"
	third      = "44444444-4444-4444-8444-444444444444"
	official   = "55555555-5555-4555-8555-555555555555"
	eventID    = "66666666-6666-4666-8666-666666666666"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	kickoff := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: home, AwayTeamID: away, Venue: "Changlimithang", Kickoff: kickoff})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.StatusScheduled || created.ID == "" {
		t.Fatalf("unexpected match %+v", created)
	}
	// Same kickoff, one of the same teams: clash.
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: third, AwayTeamID: away, Kickoff: kickoff}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("team clash: %v", err)
	}
	// Same kickoff, same venue: clash.
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: third, AwayTeamID: official, Venue: "Changlimithang", Kickoff: kickoff}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("venue clash: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: home, AwayTeamID: third, Kickoff: kickoff.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	// Overlaps within the two-hour slot clash too, and the error names what is busy.
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: third, AwayTeamID: official, Venue: "changlimithang", Kickoff: kickoff.Add(90 * time.Minute)}); !errors.Is(err, domain.ErrConflict) || err.Error() != "match scheduling conflict: venue changlimithang already has a match at 2026-11-01 10:00 UTC" {
		t.Fatalf("venue overlap (case-insensitive): %v", err)
	}
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: away, AwayTeamID: third, Kickoff: kickoff.Add(-time.Hour)}); !errors.Is(err, domain.ErrConflict) || err.Error() != "match scheduling conflict: a team already has a match at 2026-11-01 10:00 UTC" {
		t.Fatalf("team overlap before kickoff: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: third, AwayTeamID: official, Venue: "Changlimithang", Kickoff: kickoff.Add(domain.SlotLength)}); err != nil {
		t.Fatalf("a match exactly one slot later should fit: %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil || !got.Kickoff.Equal(kickoff) || got.Venue != "Changlimithang" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, official); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	page, more, err := repo.List(ctx, tournament, nil, 1, 0)
	if err != nil || len(page) != 1 || !more {
		t.Fatalf("page 1: %v %d %v", err, len(page), more)
	}
	after := kickoff.Add(time.Hour)
	page, more, err = repo.List(ctx, "", &after, 10, 0)
	if err != nil || len(page) != 2 || more {
		t.Fatalf("after filter: %v %d %v", err, len(page), more)
	}

	assigned, err := repo.AssignOfficial(ctx, created.ID, official)
	if err != nil || assigned.OfficialID != official {
		t.Fatalf("assign: %+v %v", assigned, err)
	}
	// The same official cannot also take a match that overlaps this one.
	overlapping, err := repo.Create(ctx, domain.Match{TournamentID: tournament, HomeTeamID: "77777777-7777-4777-8777-777777777777", AwayTeamID: "88888888-8888-4888-8888-888888888888", Kickoff: kickoff.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AssignOfficial(ctx, overlapping.ID, official); !errors.Is(err, domain.ErrConflict) || err.Error() != "match scheduling conflict: the official already has a match at 2026-11-01 10:00 UTC" {
		t.Fatalf("official overlap: %v", err)
	}
	if again, err := repo.AssignOfficial(ctx, created.ID, official); err != nil || again.OfficialID != official {
		t.Fatalf("re-assigning the same match must not clash with itself: %v", err)
	}
	if _, err := repo.AssignOfficial(ctx, third, official); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("assign missing: %v", err)
	}

	applied, err := repo.MarkCompleted(ctx, eventID, created.ID)
	if err != nil || !applied {
		t.Fatalf("complete: %v %v", applied, err)
	}
	if applied, err = repo.MarkCompleted(ctx, eventID, created.ID); err != nil || applied {
		t.Fatalf("duplicate event must be ignored: %v %v", applied, err)
	}
	if _, err := repo.MarkCompleted(ctx, third, official); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown match: %v", err)
	}
	if _, err := repo.AssignOfficial(ctx, created.ID, official); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("assign to completed match: %v", err)
	}
	if _, err := repo.AssignOfficial(ctx, overlapping.ID, official); err != nil {
		t.Fatalf("a completed match no longer blocks the official: %v", err)
	}
	if !IsInvalidUUID(errors.New(`invalid input syntax for type uuid: "x"`)) || IsInvalidUUID(nil) {
		t.Fatal("uuid error detection wrong")
	}
}
