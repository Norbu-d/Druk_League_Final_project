package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/search/internal/domain"
)

func TestRepositorySearchFiltersAndPaginates(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	ctx := context.Background()
	baseTime := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	rows := []struct {
		kind, id, content string
		updated           time.Time
	}{
		{"team", "11111111-1111-4111-8111-111111111111", `{"title":"Thimphu FC","subtitle":"Thimphu"}`, baseTime.Add(-3 * time.Minute)},
		{"player", "22222222-2222-4222-8222-222222222222", `{"title":"Tenzin","subtitle":"Thimphu FC"}`, baseTime.Add(-2 * time.Minute)},
		{"tournament", "33333333-3333-4333-8333-333333333333", `{"title":"Thimphu Cup","subtitle":"2026"}`, baseTime.Add(-time.Minute)},
		{"team", "44444444-4444-4444-8444-444444444444", `plain Thimphu content`, baseTime.Add(-4 * time.Minute)},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO search_documents(entity_type,entity_id,content,updated_at) VALUES($1,$2::uuid,$3,$4)`, row.kind, row.id, row.content, row.updated); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repo.Search(ctx, "thimphu", nil, 10, 0)
	if err != nil || len(all) != 4 {
		t.Fatalf("unfiltered hits = %v, err=%v", all, err)
	}
	if all[0].EntityID != rows[2].id {
		t.Fatalf("first result = %+v, want most recently updated tournament", all[0])
	}
	byID := make(map[string]domain.Hit, len(all))
	for _, hit := range all {
		byID[hit.EntityID] = hit
	}
	if hit := byID[rows[0].id]; hit.Title != "Thimphu FC" || hit.Subtitle != "Thimphu" {
		t.Fatalf("JSON display fields = %+v", hit)
	}
	if hit := byID[rows[3].id]; hit.Title != "plain Thimphu content" || hit.Subtitle != "" {
		t.Fatalf("plain content fallback = %+v", hit)
	}

	teams, err := repo.Search(ctx, "THIMPHU", []string{"team"}, 10, 0)
	if err != nil || len(teams) != 2 {
		t.Fatalf("team filter hits = %v, err=%v", teams, err)
	}
	page, err := repo.Search(ctx, "thimphu", []string{"team", "player"}, 1, 1)
	if err != nil || len(page) != 1 {
		t.Fatalf("multi-type page = %v, err=%v", page, err)
	}
	if _, err := repo.Search(ctx, "thimphu", []string{"team", "player", "tournament"}, 10, 0); err != nil {
		t.Fatalf("three-type search: %v", err)
	}
}

func TestRepositoryRejectsTooManyFiltersAndReturnsDatabaseErrors(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	if _, err := repo.Search(context.Background(), "x", []string{"a", "b", "c", "d"}, 10, 0); err == nil {
		t.Fatal("expected too many filters error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.Search(ctx, "x", nil, 10, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}
