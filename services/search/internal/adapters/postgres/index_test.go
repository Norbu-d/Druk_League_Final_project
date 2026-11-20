package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/search/internal/domain"
)

func TestApplyIndexEventIsIdempotentVersionedAndKeepsDeleteTombstone(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	ctx := context.Background()
	teamID := "11111111-1111-4111-8111-111111111111"
	event := domain.IndexEvent{
		EventID: "22222222-2222-4222-8222-222222222222", EntityType: domain.EntityTeam,
		EntityID: teamID, Version: 1, Operation: domain.OperationUpsert,
		OccurredAt: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC),
		Fields:     domain.DocumentFields{Title: "Thimphu FC", Subtitle: "Thimphu", SearchText: "Thimphu FC"},
	}
	if err := repo.ApplyIndexEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := repo.ApplyIndexEvent(ctx, event); err != nil {
		t.Fatalf("duplicate event: %v", err)
	}
	update := event
	update.EventID = "33333333-3333-4333-8333-333333333333"
	update.Version = 2
	update.Fields.Title = "Thimphu United"
	if err := repo.ApplyIndexEvent(ctx, update); err != nil {
		t.Fatal(err)
	}
	stale := event
	stale.EventID = "44444444-4444-4444-8444-444444444444"
	stale.Fields.Title = "Stale Team Name"
	if err := repo.ApplyIndexEvent(ctx, stale); err != nil {
		t.Fatalf("stale event should be acknowledged: %v", err)
	}
	var content string
	var version int64
	if err := db.QueryRowContext(ctx, `SELECT content,version FROM search_documents WHERE entity_type='team' AND entity_id=$1`, teamID).Scan(&content, &version); err != nil {
		t.Fatal(err)
	}
	var fields domain.DocumentFields
	if err := json.Unmarshal([]byte(content), &fields); err != nil {
		t.Fatal(err)
	}
	if fields.Title != "Thimphu United" || version != 2 {
		t.Fatalf("fields=%+v version=%d", fields, version)
	}
	remove := update
	remove.EventID = "55555555-5555-4555-8555-555555555555"
	remove.Version = 3
	remove.Operation = domain.OperationDelete
	remove.Fields = domain.DocumentFields{}
	if err := repo.ApplyIndexEvent(ctx, remove); err != nil {
		t.Fatal(err)
	}
	var documentCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM search_documents WHERE entity_type='team' AND entity_id=$1`, teamID).Scan(&documentCount); err != nil {
		t.Fatal(err)
	}
	var deleted bool
	if err := db.QueryRowContext(ctx, `SELECT deleted FROM search_entity_versions WHERE entity_type='team' AND entity_id=$1`, teamID).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if documentCount != 0 || !deleted {
		t.Fatalf("documentCount=%d deleted=%v", documentCount, deleted)
	}
}
