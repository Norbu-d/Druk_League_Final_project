package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/score/internal/domain"
)

const (
	matchID  = "11111111-1111-4111-8111-111111111111"
	official = "22222222-2222-4222-8222-222222222222"
	eventID  = "33333333-3333-4333-8333-333333333333"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	ctx := context.Background()

	if _, err := repo.Get(ctx, matchID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty: %v", err)
	}
	live, err := repo.UpdateLive(ctx, matchID, 2, 1)
	if err != nil || live.HomeScore != 2 || live.Finalized {
		t.Fatalf("live: %+v %v", live, err)
	}
	match := domain.Match{ID: matchID, TournamentID: matchID, HomeTeamID: official, AwayTeamID: eventID}
	final, err := repo.Finalize(ctx, match, eventID, official, "full time")
	if err != nil || !final.Finalized || final.HomeScore != 2 {
		t.Fatalf("finalize: %+v %v", final, err)
	}
	if _, err := repo.Finalize(ctx, match, eventID, official, ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second finalize: %v", err)
	}
	if _, err := repo.UpdateLive(ctx, matchID, 5, 5); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("update after final: %v", err)
	}
	// The immutability trigger blocks direct edits too.
	if _, err := db.ExecContext(ctx, `UPDATE scores SET home_score=9 WHERE match_id=$1`, matchID); err == nil {
		t.Fatal("trigger did not block edit of a finalized score")
	}
	got, err := repo.Get(ctx, matchID)
	if err != nil || !got.Finalized || got.HomeScore != 2 {
		t.Fatalf("get: %+v %v", got, err)
	}

	pending, err := repo.PendingEvents(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].EventID != eventID {
		t.Fatalf("pending: %+v %v", pending, err)
	}
	var event domain.MatchFinalized
	if err := json.Unmarshal(pending[0].Payload, &event); err != nil || event.FinalizedBy != official || event.Reason != "full time" || event.HomeScore != 2 {
		t.Fatalf("event payload: %+v %v", event, err)
	}
	if err := repo.MarkPublished(ctx, eventID); err != nil {
		t.Fatal(err)
	}
	if pending, _ = repo.PendingEvents(ctx, 10); len(pending) != 0 {
		t.Fatal("published event still pending")
	}
}
