package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/services/match/internal/domain"
)

const (
	tournament = "11111111-1111-4111-8111-111111111111"
	home       = "22222222-2222-4222-8222-222222222222"
	away       = "33333333-3333-4333-8333-333333333333"
	official   = "44444444-4444-4444-8444-444444444444"
	matchID    = "55555555-5555-4555-8555-555555555555"
	eventID    = "66666666-6666-4666-8666-666666666666"
)

type fakeRepo struct {
	matches   map[string]domain.Match
	created   []domain.Match
	completed map[string]bool
	err       error
}

func newRepo() *fakeRepo {
	return &fakeRepo{matches: map[string]domain.Match{matchID: {ID: matchID, Status: domain.StatusScheduled}}, completed: map[string]bool{}}
}
func (f *fakeRepo) Create(_ context.Context, m domain.Match) (domain.Match, error) {
	if f.err != nil {
		return domain.Match{}, f.err
	}
	m.ID = "new"
	f.created = append(f.created, m)
	return m, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Match, error) {
	m, ok := f.matches[id]
	if !ok {
		return domain.Match{}, domain.ErrNotFound
	}
	return m, nil
}
func (f *fakeRepo) List(context.Context, string, *time.Time, int, int) ([]domain.Match, bool, error) {
	return []domain.Match{{ID: matchID}}, false, nil
}
func (f *fakeRepo) AssignOfficial(_ context.Context, id, officialID string) (domain.Match, error) {
	m := f.matches[id]
	m.OfficialID = officialID
	return m, nil
}
func (f *fakeRepo) MarkCompleted(_ context.Context, event, _ string) (bool, error) {
	if f.completed[event] {
		return false, nil
	}
	f.completed[event] = true
	return true, nil
}

type fakeTeams map[string]string

func (f fakeTeams) TournamentForTeam(_ context.Context, id string) (string, error) {
	if id == "down" {
		return "", errors.New("unavailable")
	}
	tournamentID, ok := f[id]
	if !ok {
		return "", domain.ErrInvalid
	}
	return tournamentID, nil
}

type fakeUsers struct {
	role string
	err  error
}

func (f fakeUsers) RoleForUser(context.Context, string) (string, error) { return f.role, f.err }

func service(repo *fakeRepo, users fakeUsers) *Service {
	return New(repo, fakeTeams{home: tournament, away: tournament}, users)
}

func validMatch() domain.Match {
	return domain.Match{TournamentID: " " + tournament, HomeTeamID: home, AwayTeamID: away, Venue: " Changlimithang ", Kickoff: time.Now()}
}

func TestCreateSchedulesValidMatch(t *testing.T) {
	repo := newRepo()
	created, err := service(repo, fakeUsers{}).Create(context.Background(), "super_admin", validMatch())
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.StatusScheduled || created.Venue != "Changlimithang" || created.TournamentID != tournament {
		t.Fatalf("unexpected match %+v", created)
	}
}

func TestCreateRejections(t *testing.T) {
	cases := map[string]struct {
		role   string
		mutate func(*domain.Match)
		want   error
	}{
		"viewer":           {"viewer", func(*domain.Match) {}, domain.ErrForbidden},
		"same team":        {"super_admin", func(m *domain.Match) { m.AwayTeamID = home }, domain.ErrInvalid},
		"bad uuid":         {"super_admin", func(m *domain.Match) { m.HomeTeamID = "x" }, domain.ErrInvalid},
		"no kickoff":       {"super_admin", func(m *domain.Match) { m.Kickoff = time.Time{} }, domain.ErrInvalid},
		"other tournament": {"super_admin", func(m *domain.Match) { m.TournamentID = official }, domain.ErrInvalid},
		"unknown team":     {"super_admin", func(m *domain.Match) { m.AwayTeamID = official }, domain.ErrInvalid},
	}
	for name, c := range cases {
		m := validMatch()
		c.mutate(&m)
		if _, err := service(newRepo(), fakeUsers{}).Create(context.Background(), c.role, m); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", name, err, c.want)
		}
	}
}

func TestCreateReportsTeamServiceOutage(t *testing.T) {
	s := New(newRepo(), fakeTeams{home: tournament, "down": tournament}, fakeUsers{})
	m := validMatch()
	m.AwayTeamID = "down"
	m.AwayTeamID = "77777777-7777-4777-8777-777777777777"
	s.teams = outageTeams{}
	if _, err := s.Create(context.Background(), "super_admin", m); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("got %v", err)
	}
}

type outageTeams struct{}

func (outageTeams) TournamentForTeam(context.Context, string) (string, error) {
	return "", domain.ErrDependency
}

func TestGetAndList(t *testing.T) {
	s := service(newRepo(), fakeUsers{})
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("invalid id accepted")
	}
	if m, err := s.Get(context.Background(), matchID); err != nil || m.ID != matchID {
		t.Fatalf("get: %v", err)
	}
	if _, _, err := s.List(context.Background(), "nope", nil, 10, 0); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("invalid tournament filter accepted")
	}
	if items, _, err := s.List(context.Background(), "", nil, 10, 0); err != nil || len(items) != 1 {
		t.Fatalf("list: %v", err)
	}
}

func TestAssignOfficial(t *testing.T) {
	repo := newRepo()
	s := service(repo, fakeUsers{role: "match_official"})
	m, err := s.AssignOfficial(context.Background(), "super_admin", matchID, official)
	if err != nil || m.OfficialID != official {
		t.Fatalf("assign: %+v %v", m, err)
	}
	cases := []struct {
		name            string
		role, id, offID string
		users           fakeUsers
		want            error
	}{
		{"viewer", "viewer", matchID, official, fakeUsers{role: "match_official"}, domain.ErrForbidden},
		{"bad ids", "super_admin", "x", official, fakeUsers{role: "match_official"}, domain.ErrInvalid},
		{"missing match", "super_admin", eventID, official, fakeUsers{role: "match_official"}, domain.ErrNotFound},
		{"not an official", "super_admin", matchID, official, fakeUsers{role: "viewer"}, domain.ErrInvalid},
		{"unknown user", "super_admin", matchID, official, fakeUsers{err: domain.ErrInvalid}, domain.ErrInvalid},
		{"user service down", "super_admin", matchID, official, fakeUsers{err: errors.New("down")}, domain.ErrDependency},
	}
	for _, c := range cases {
		if _, err := service(newRepo(), c.users).AssignOfficial(context.Background(), c.role, c.id, c.offID); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
	}
	completed := newRepo()
	completed.matches[matchID] = domain.Match{ID: matchID, Status: domain.StatusCompleted}
	if _, err := service(completed, fakeUsers{role: "match_official"}).AssignOfficial(context.Background(), "super_admin", matchID, official); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("completed match: got %v", err)
	}
}

func TestMarkCompletedIsIdempotent(t *testing.T) {
	s := service(newRepo(), fakeUsers{})
	if applied, err := s.MarkCompleted(context.Background(), eventID, matchID); err != nil || !applied {
		t.Fatalf("first delivery: %v %v", applied, err)
	}
	if applied, err := s.MarkCompleted(context.Background(), eventID, matchID); err != nil || applied {
		t.Fatalf("duplicate delivery must be ignored: %v %v", applied, err)
	}
	if _, err := s.MarkCompleted(context.Background(), "", matchID); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("missing event id accepted")
	}
}
