package app

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/drukleague/platform/services/search/internal/domain"
)

type fakeRepository struct {
	hits   []domain.Hit
	err    error
	query  string
	types  []string
	limit  int
	offset int
}

func (r *fakeRepository) Search(_ context.Context, query string, types []string, limit, offset int) ([]domain.Hit, error) {
	r.query, r.types, r.limit, r.offset = query, types, limit, offset
	return r.hits, r.err
}

func TestSearchNormalizesFiltersAndBuildsNextPageToken(t *testing.T) {
	repo := &fakeRepository{hits: []domain.Hit{{Title: "one"}, {Title: "two"}, {Title: "extra"}}}
	page, err := New(repo).Search(context.Background(), domain.SearchRequest{
		Query: "  Finals  ", EntityTypes: []string{" TEAM ", "player", "team"}, PageSize: 2, PageToken: "4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.query != "Finals" || !reflect.DeepEqual(repo.types, []string{"team", "player"}) || repo.limit != 3 || repo.offset != 4 {
		t.Fatalf("repository call = query %q, types %v, limit %d, offset %d", repo.query, repo.types, repo.limit, repo.offset)
	}
	if len(page.Hits) != 2 || page.NextPageToken != "6" {
		t.Fatalf("page = %+v, want 2 hits and next token 6", page)
	}
}

func TestSearchDefaultsAndCapsPageSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
		want int
	}{{"default", 0, defaultPageSize}, {"negative", -1, defaultPageSize}, {"cap", maximumPageSize + 50, maximumPageSize}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := New(repo).Search(context.Background(), domain.SearchRequest{Query: "x", PageSize: tc.size})
			if err != nil {
				t.Fatal(err)
			}
			if repo.limit != tc.want+1 {
				t.Fatalf("repository limit = %d, want %d", repo.limit, tc.want+1)
			}
		})
	}
}

func TestSearchRejectsInvalidRequestsWithoutQueryingRepository(t *testing.T) {
	longQuery := strings.Repeat("界", maximumQueryLen+1)
	for _, tc := range []struct {
		name    string
		request domain.SearchRequest
	}{
		{"empty", domain.SearchRequest{Query: " \t"}},
		{"query too long by rune count", domain.SearchRequest{Query: longQuery}},
		{"unknown entity type", domain.SearchRequest{Query: "x", EntityTypes: []string{"venue"}}},
		{"invalid page token", domain.SearchRequest{Query: "x", PageToken: "1.5"}},
		{"negative page token", domain.SearchRequest{Query: "x", PageToken: "-1"}},
		{"offset too large", domain.SearchRequest{Query: "x", PageToken: "1000001"}},
		{"offset plus size too large", domain.SearchRequest{Query: "x", PageSize: 100, PageToken: "999901"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := New(repo).Search(context.Background(), tc.request)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want ErrInvalidRequest", err)
			}
			if repo.query != "" {
				t.Fatal("repository was called for invalid request")
			}
		})
	}
}

func TestSearchWrapsRepositoryError(t *testing.T) {
	want := errors.New("database unavailable")
	_, err := New(&fakeRepository{err: want}).Search(context.Background(), domain.SearchRequest{Query: "x"})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped repository error", err)
	}
}
