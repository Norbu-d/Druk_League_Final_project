package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/services/standings/internal/domain"
)

const (
	eventID    = "11111111-1111-4111-8111-111111111111"
	matchID    = "22222222-2222-4222-8222-222222222222"
	tournament = "33333333-3333-4333-8333-333333333333"
	home       = "44444444-4444-4444-8444-444444444444"
	away       = "55555555-5555-4555-8555-555555555555"
)

type fakeRepo struct {
	known   map[string]bool
	applied []domain.Team
	err     error
}

func (f *fakeRepo) TeamKnown(_ context.Context, _, team string) (bool, error) {
	return f.known[team], f.err
}
func (f *fakeRepo) ApplyFinalized(_ context.Context, _ domain.MatchFinalized, h, a domain.Team) (bool, error) {
	f.applied = append(f.applied, h, a)
	return true, nil
}
func (f *fakeRepo) Get(context.Context, string) ([]domain.Standing, error) {
	return []domain.Standing{{TeamID: home, Points: 3}}, nil
}

type fakeTeams struct {
	teams map[string]domain.Team
	err   error
	calls int
}

func (f *fakeTeams) GetTeam(_ context.Context, id string) (domain.Team, error) {
	f.calls++
	if f.err != nil {
		return domain.Team{}, f.err
	}
	return f.teams[id], nil
}

func directory() *fakeTeams {
	return &fakeTeams{teams: map[string]domain.Team{
		home: {ID: home, Name: "Thimphu FC", Dzongkhag: "Thimphu", TournamentID: tournament},
		away: {ID: away, Name: "Paro FC", Dzongkhag: "Paro", TournamentID: tournament},
	}}
}

func event() domain.MatchFinalized {
	return domain.MatchFinalized{EventID: eventID, MatchID: matchID, TournamentID: tournament, HomeTeamID: home, AwayTeamID: away, HomeScore: 2, AwayScore: 1}
}

func TestApplyFinalizedLooksUpUnknownTeams(t *testing.T) {
	repo, teams := &fakeRepo{known: map[string]bool{}}, directory()
	if _, err := New(repo, teams).ApplyFinalized(context.Background(), event()); err != nil {
		t.Fatal(err)
	}
	if teams.calls != 2 || repo.applied[0].Name != "Thimphu FC" || repo.applied[1].Dzongkhag != "Paro" {
		t.Fatalf("unexpected lookups %d %+v", teams.calls, repo.applied)
	}
}

func TestApplyFinalizedSkipsLookupForKnownTeams(t *testing.T) {
	repo, teams := &fakeRepo{known: map[string]bool{home: true, away: true}}, directory()
	if _, err := New(repo, teams).ApplyFinalized(context.Background(), event()); err != nil || teams.calls != 0 {
		t.Fatalf("known teams should not be looked up: %v %d", err, teams.calls)
	}
}

func TestApplyFinalizedRejections(t *testing.T) {
	invalid := []func(*domain.MatchFinalized){
		func(e *domain.MatchFinalized) { e.EventID = "" },
		func(e *domain.MatchFinalized) { e.AwayTeamID = home },
		func(e *domain.MatchFinalized) { e.HomeScore = -1 },
		func(e *domain.MatchFinalized) { e.TournamentID = "x" },
	}
	for i, mutate := range invalid {
		e := event()
		mutate(&e)
		if _, err := New(&fakeRepo{known: map[string]bool{}}, directory()).ApplyFinalized(context.Background(), e); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("case %d: got %v", i, err)
		}
	}
	// Team in a different tournament.
	teams := directory()
	teams.teams[away] = domain.Team{ID: away, Name: "Paro FC", Dzongkhag: "Paro", TournamentID: matchID}
	if _, err := New(&fakeRepo{known: map[string]bool{}}, teams).ApplyFinalized(context.Background(), event()); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("foreign team: %v", err)
	}
	if _, err := New(&fakeRepo{known: map[string]bool{}}, &fakeTeams{err: domain.ErrInvalid}).ApplyFinalized(context.Background(), event()); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("unknown team: %v", err)
	}
	if _, err := New(&fakeRepo{known: map[string]bool{}}, &fakeTeams{err: errors.New("down")}).ApplyFinalized(context.Background(), event()); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("team service down: %v", err)
	}
	if _, err := New(&fakeRepo{err: errors.New("db")}, directory()).ApplyFinalized(context.Background(), event()); err == nil {
		t.Fatal("repository error ignored")
	}
}

func TestGet(t *testing.T) {
	s := New(&fakeRepo{}, directory())
	if _, err := s.Get(context.Background(), ""); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("empty tournament accepted")
	}
	if rows, err := s.Get(context.Background(), tournament); err != nil || len(rows) != 1 {
		t.Fatalf("get: %v", err)
	}
}

func TestOutcome(t *testing.T) {
	cases := []struct{ gf, ga, w, d, l, p int32 }{{2, 1, 1, 0, 0, 3}, {1, 1, 0, 1, 0, 1}, {0, 3, 0, 0, 1, 0}}
	for _, c := range cases {
		w, d, l, p := domain.Outcome(c.gf, c.ga)
		if w != c.w || d != c.d || l != c.l || p != c.p {
			t.Fatalf("%d-%d: got %d %d %d %d", c.gf, c.ga, w, d, l, p)
		}
	}
}
