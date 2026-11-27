package grpcadapter

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/dzongkhag/v1"
	"github.com/drukleague/platform/services/dzongkhag/internal/app"
	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRepo struct {
	items []domain.Dzongkhag
	err   error
}

func (f fakeRepo) Get(_ context.Context, code string) (domain.Dzongkhag, error) {
	if f.err != nil {
		return domain.Dzongkhag{}, f.err
	}
	for _, d := range f.items {
		if d.Code == code {
			return d, nil
		}
	}
	return domain.Dzongkhag{}, domain.ErrNotFound
}

func (f fakeRepo) List(context.Context) ([]domain.Dzongkhag, error) { return f.items, f.err }

func newServer(repo fakeRepo) *Server { return New(app.New(repo)) }

var paro = domain.Dzongkhag{Code: "08", Name: "Paro"}

func TestGetDzongkhag(t *testing.T) {
	s := newServer(fakeRepo{items: []domain.Dzongkhag{paro}})
	got, err := s.GetDzongkhag(context.Background(), &v1.GetDzongkhagRequest{Code: "08"})
	if err != nil || got.GetDzongkhag().GetCode() != "08" || got.GetDzongkhag().GetName() != "Paro" {
		t.Fatalf("get: %v %v", got, err)
	}
	if _, err := s.GetDzongkhag(context.Background(), &v1.GetDzongkhagRequest{Code: "99"}); status.Code(err) != codes.NotFound {
		t.Fatalf("unknown code: %v", err)
	}
	broken := newServer(fakeRepo{err: errors.New("database unavailable")})
	if _, err := broken.GetDzongkhag(context.Background(), &v1.GetDzongkhagRequest{Code: "08"}); status.Code(err) != codes.Internal {
		t.Fatalf("repository failure: %v", err)
	}
}

func TestListDzongkhags(t *testing.T) {
	s := newServer(fakeRepo{items: []domain.Dzongkhag{paro, {Code: "14", Name: "Thimphu"}}})
	got, err := s.ListDzongkhags(context.Background(), &v1.ListDzongkhagsRequest{})
	if err != nil || len(got.GetDzongkhags()) != 2 || got.GetDzongkhags()[1].GetName() != "Thimphu" {
		t.Fatalf("list: %v %v", got, err)
	}
	empty, err := newServer(fakeRepo{}).ListDzongkhags(context.Background(), &v1.ListDzongkhagsRequest{})
	if err != nil || empty.GetDzongkhags() == nil || len(empty.GetDzongkhags()) != 0 {
		t.Fatalf("empty list: %v %v", empty, err)
	}
	if _, err := newServer(fakeRepo{err: errors.New("database unavailable")}).ListDzongkhags(context.Background(), &v1.ListDzongkhagsRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("repository failure: %v", err)
	}
}
