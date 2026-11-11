package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

func TestRepositoryRecordsDeduplicatesAndListsEntries(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	when := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	first, err := repo.Record(ctx, domain.Entry{EventID: "11111111-1111-4111-8111-111111111111", ActorID: "admin", Action: "team.updated", ResourceType: "team", ResourceID: "team-1", Reason: "corrected", OccurredAt: when})
	if err != nil || first.ID == "" || first.EventID == "" {
		t.Fatalf("first record = %+v, err=%v", first, err)
	}
	duplicate, err := repo.Record(ctx, domain.Entry{EventID: first.EventID, ActorID: "other", Action: "changed", ResourceType: "other", ResourceID: "other", OccurredAt: when.Add(time.Hour)})
	if err != nil || duplicate.ID != first.ID || duplicate.ActorID != first.ActorID || duplicate.Action != first.Action {
		t.Fatalf("duplicate = %+v, err=%v; want original %+v", duplicate, err, first)
	}

	second, err := repo.Record(ctx, domain.Entry{ActorID: "official", Action: "match.finalized", ResourceType: "match", ResourceID: "match-1", OccurredAt: when.Add(time.Minute)})
	if err != nil || second.ID == "" || second.EventID != "" {
		t.Fatalf("second record = %+v, err=%v", second, err)
	}

	items, more, err := repo.List(ctx, "", "", 1, 0)
	if err != nil || len(items) != 1 || !more {
		t.Fatalf("first page = %v, more=%v err=%v", items, more, err)
	}
	items, more, err = repo.List(ctx, "team", "team-1", 10, 0)
	if err != nil || len(items) != 1 || more || items[0].ID != first.ID || items[0].Reason != "corrected" {
		t.Fatalf("filtered list = %v, more=%v err=%v", items, more, err)
	}
	items, more, err = repo.List(ctx, "match", "match-1", 10, 0)
	if err != nil || len(items) != 1 || more || items[0].ID != second.ID {
		t.Fatalf("match list = %v, more=%v err=%v", items, more, err)
	}
}

func TestRepositoryReturnsDatabaseErrors(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := repo.Record(ctx, domain.Entry{ActorID: "admin", Action: "team.updated", ResourceType: "team", ResourceID: "team-1", OccurredAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("record with canceled context succeeded")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("record error = %v, want context canceled", err)
	}
	if _, _, err := repo.List(ctx, "", "", 10, 0); err == nil {
		t.Fatal("list with canceled context succeeded")
	}
}
