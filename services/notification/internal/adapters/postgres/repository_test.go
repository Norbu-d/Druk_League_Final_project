package postgres

import (
	"context"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	item := domain.Notification{EventID: "11111111-1111-4111-8111-111111111111", UserID: "22222222-2222-4222-8222-222222222222", Channel: "in_app", Subject: "Result finalized", Body: "Thimphu FC 2–1 Paro FC", Status: "pending"}
	created, inserted, err := repo.Create(ctx, item)
	if err != nil || !inserted || created.ID == "" || created.EventID != item.EventID || created.UserID != item.UserID {
		t.Fatalf("create: item=%+v inserted=%v err=%v", created, inserted, err)
	}
	duplicate, inserted, err := repo.Create(ctx, item)
	if err != nil || inserted || duplicate.ID != created.ID {
		t.Fatalf("idempotent create: item=%+v inserted=%v err=%v", duplicate, inserted, err)
	}
	second := item
	second.EventID = "33333333-3333-4333-8333-333333333333"
	second.UserID = "44444444-4444-4444-8444-444444444444"
	if _, inserted, err := repo.Create(ctx, second); err != nil || !inserted {
		t.Fatalf("second create: inserted=%v err=%v", inserted, err)
	}
	page, more, err := repo.List(ctx, item.UserID, 1, 0)
	if err != nil || more || len(page) != 1 || page[0].ID != created.ID || page[0].Status != "pending" {
		t.Fatalf("first page: items=%+v more=%v err=%v", page, more, err)
	}
	empty, more, err := repo.List(ctx, "55555555-5555-4555-8555-555555555555", 10, 0)
	if err != nil || more || len(empty) != 0 {
		t.Fatalf("empty list: items=%+v more=%v err=%v", empty, more, err)
	}
}
