package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
)

type fakeRepo struct {
	items map[string]domain.Dzongkhag
	err   error
}

func (f fakeRepo) Get(_ context.Context, code string) (domain.Dzongkhag, error) {
	if f.err != nil {
		return domain.Dzongkhag{}, f.err
	}
	d, ok := f.items[code]
	if !ok {
		return domain.Dzongkhag{}, domain.ErrNotFound
	}
	return d, nil
}

func (f fakeRepo) List(context.Context) ([]domain.Dzongkhag, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make([]domain.Dzongkhag, 0, len(f.items))
	for _, d := range f.items {
		result = append(result, d)
	}
	return result, nil
}

func TestGet(t *testing.T) {
	s := New(fakeRepo{items: map[string]domain.Dzongkhag{"14": {Code: "14", Name: "Thimphu"}}})
	got, err := s.Get(context.Background(), "14")
	if err != nil || got.Name != "Thimphu" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := s.Get(context.Background(), "99"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown code: got %v", err)
	}
}

func TestList(t *testing.T) {
	s := New(fakeRepo{items: map[string]domain.Dzongkhag{"08": {Code: "08", Name: "Paro"}, "14": {Code: "14", Name: "Thimphu"}}})
	got, err := s.List(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("list: %+v %v", got, err)
	}
}

func TestRepositoryErrorsPassThrough(t *testing.T) {
	boom := errors.New("database unavailable")
	s := New(fakeRepo{err: boom})
	if _, err := s.Get(context.Background(), "14"); !errors.Is(err, boom) {
		t.Fatalf("get: got %v", err)
	}
	if _, err := s.List(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("list: got %v", err)
	}
}
