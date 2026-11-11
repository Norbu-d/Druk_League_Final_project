package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 20 {
		t.Fatalf("Bhutan has 20 dzongkhags, got %d", len(all))
	}
	if all[0].Name != "Bumthang" || all[len(all)-1].Name != "Zhemgang" {
		t.Fatalf("list should be ordered by name: first %q, last %q", all[0].Name, all[len(all)-1].Name)
	}

	for code, name := range map[string]string{"01": "Bumthang", "08": "Paro", "14": "Thimphu", "20": "Zhemgang"} {
		got, err := repo.Get(ctx, code)
		if err != nil || got.Code != code || got.Name != name {
			t.Fatalf("get %s: %+v %v", code, got, err)
		}
	}
	for _, code := range []string{"", "00", "21", "99", "thimphu"} {
		if _, err := repo.Get(ctx, code); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("get %q: got %v, want ErrNotFound", code, err)
		}
	}
}

func TestRepositoryReportsDatabaseFailures(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	// A cancelled context makes every query fail, which is how a lost connection looks to the repository.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.Get(ctx, "14"); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get should fail with a real error, got %v", err)
	}
	if _, err := repo.List(ctx); err == nil {
		t.Fatal("list should fail")
	}
}
