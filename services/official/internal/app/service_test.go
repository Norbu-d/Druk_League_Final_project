package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/official/internal/domain"
)

const (
	matchID    = "11111111-1111-4111-8111-111111111111"
	official   = "22222222-2222-4222-8222-222222222222"
	tournament = "33333333-3333-4333-8333-333333333333"
	adminID    = "44444444-4444-4444-8444-444444444444"
)

type fakeRepo struct {
	recorded []domain.Assignment
	filters  [2]string
}

func (f *fakeRepo) Record(_ context.Context, a domain.Assignment) (domain.Assignment, error) {
	f.recorded = append(f.recorded, a)
	return a, nil
}
func (f *fakeRepo) List(_ context.Context, officialID, tournamentID string) ([]domain.Assignment, error) {
	f.filters = [2]string{officialID, tournamentID}
	return []domain.Assignment{{MatchID: matchID}}, nil
}

type fakeMatches struct{ err error }

func (f fakeMatches) AssignOfficial(_ context.Context, matchID, officialID string) (domain.MatchRef, error) {
	return domain.MatchRef{ID: matchID, TournamentID: tournament, OfficialID: officialID}, f.err
}

var admin = Caller{ID: adminID, Role: core.RoleSuperAdmin}

func TestAssignRecordsTournamentAndActor(t *testing.T) {
	repo := &fakeRepo{}
	a, err := New(repo, fakeMatches{}).Assign(context.Background(), admin, " "+matchID+" ", official)
	if err != nil {
		t.Fatal(err)
	}
	if a.TournamentID != tournament || a.AssignedBy != adminID || a.MatchID != matchID {
		t.Fatalf("unexpected assignment %+v", a)
	}
}

func TestAssignRejections(t *testing.T) {
	cases := []struct {
		name    string
		caller  Caller
		match   string
		matches fakeMatches
		want    error
	}{
		{"official cannot assign", Caller{ID: official, Role: core.RoleMatchOfficial}, matchID, fakeMatches{}, core.ErrForbidden},
		{"bad ids", admin, "x", fakeMatches{}, core.ErrInvalid},
		{"match service refuses", admin, matchID, fakeMatches{err: core.Conflict("completed")}, core.ErrConflict},
	}
	for _, c := range cases {
		repo := &fakeRepo{}
		if _, err := New(repo, c.matches).Assign(context.Background(), c.caller, c.match, official); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v", c.name, err)
		}
		if len(repo.recorded) != 0 {
			t.Fatalf("%s: nothing should be recorded", c.name)
		}
	}
}

func TestListScopes(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, fakeMatches{})
	if _, err := s.List(context.Background(), Caller{ID: official, Role: core.RoleMatchOfficial}, "", ""); err != nil || repo.filters[0] != official {
		t.Fatalf("official defaults to self: %v %v", err, repo.filters)
	}
	if _, err := s.List(context.Background(), Caller{ID: official, Role: core.RoleMatchOfficial}, adminID, ""); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("official listed someone else")
	}
	if _, err := s.List(context.Background(), Caller{ID: "v", Role: core.RoleViewer}, "", ""); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("viewer listed assignments")
	}
	if _, err := s.List(context.Background(), admin, "", tournament); err != nil || repo.filters != [2]string{"", tournament} {
		t.Fatalf("admin tournament filter: %v %v", err, repo.filters)
	}
	if _, err := s.List(context.Background(), admin, "", "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad filter accepted")
	}
}
