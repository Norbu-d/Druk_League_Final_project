package grpcadapter

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/search/v1"
	"github.com/drukleague/platform/services/search/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRepository struct {
	hits          []domain.Hit
	err           error
	query         string
	types         []string
	limit, offset int
}

func (r *fakeRepository) Search(_ context.Context, query string, types []string, limit, offset int) ([]domain.Hit, error) {
	r.query, r.types, r.limit, r.offset = query, types, limit, offset
	return r.hits, r.err
}

func TestSearchMapsRequestAndHits(t *testing.T) {
	repo := &fakeRepository{hits: []domain.Hit{{EntityType: "team", EntityID: "id-1", Title: "Thimphu FC", Subtitle: "Thimphu"}}}
	response, err := New(repo).Search(context.Background(), &v1.SearchRequest{Query: " Thimphu ", EntityTypes: []string{"team"}, PageSize: 5, PageToken: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.query != "Thimphu" || len(repo.types) != 1 || repo.types[0] != "team" || repo.limit != 6 || repo.offset != 10 {
		t.Fatalf("repository call = %+v", repo)
	}
	if len(response.GetHits()) != 1 || response.GetHits()[0].GetEntityId() != "id-1" || response.GetHits()[0].GetTitle() != "Thimphu FC" || response.GetHits()[0].GetSubtitle() != "Thimphu" {
		t.Fatalf("response = %+v", response)
	}
}

func TestSearchRejectsNilAndInvalidRequests(t *testing.T) {
	server := New(&fakeRepository{})
	if _, err := server.Search(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request code = %s", status.Code(err))
	}
	if _, err := server.Search(context.Background(), &v1.SearchRequest{Query: " ", EntityTypes: []string{"venue"}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid request code = %s", status.Code(err))
	}
}

func TestSearchMapsRepositoryErrorsToGrpcStatus(t *testing.T) {
	_, err := New(&fakeRepository{err: errors.New("database unavailable")}).Search(context.Background(), &v1.SearchRequest{Query: "team"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %s, want Internal (error %v)", status.Code(err), err)
	}
}
