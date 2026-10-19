package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

const (
	tournament = "11111111-1111-4111-8111-111111111111"
	adminID    = "22222222-2222-4222-8222-222222222222"
)

var admin = Caller{ID: adminID, Role: core.RoleSuperAdmin}

type fakeRepo struct {
	stored  []domain.Fixture
	event   domain.OutboxEvent
	linked  map[string]string
	linkErr error
}

func (f *fakeRepo) SaveSchedule(_ context.Context, _ string, fixtures []domain.Fixture, event domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	if f.stored != nil {
		return append([]domain.Fixture(nil), f.stored...), false, nil
	}
	for i := range fixtures {
		fixtures[i].ID = string(rune('a' + i))
	}
	f.stored, f.event = fixtures, event
	return append([]domain.Fixture(nil), fixtures...), true, nil
}
func (f *fakeRepo) LinkMatch(_ context.Context, fixtureID, matchID string) error {
	if f.linkErr != nil {
		return f.linkErr
	}
	f.linked[fixtureID] = matchID
	for i := range f.stored {
		if f.stored[i].ID == fixtureID {
			f.stored[i].MatchID = matchID
		}
	}
	return nil
}
func (f *fakeRepo) List(context.Context, string, int, int) ([]domain.Fixture, bool, error) {
	return f.stored, false, nil
}

type fakeTournaments struct{ err error }

func (f fakeTournaments) Get(context.Context, string) (domain.Tournament, error) {
	return domain.Tournament{ID: tournament, Dzongkhag: "Thimphu"}, f.err
}

type fakeTeams struct {
	ids []string
	err error
}

func (f fakeTeams) IDsInTournament(context.Context, string) ([]string, error) { return f.ids, f.err }

type fakeVenues struct{ dzongkhag *string }

func (f fakeVenues) InDzongkhag(_ context.Context, dzongkhag string) ([]domain.Venue, error) {
	*f.dzongkhag = dzongkhag
	return []domain.Venue{{ID: "v1", Name: "Changlimithang"}}, nil
}

type fakeMatches struct {
	created int
	failAt  int
}

func (f *fakeMatches) Create(context.Context, domain.Fixture) (string, error) {
	f.created++
	if f.failAt > 0 && f.created == f.failAt {
		return "", core.Dependency("match service down")
	}
	return "match-" + string(rune('0'+f.created)), nil
}

func newService(repo *fakeRepo, teamIDs []string, matches *fakeMatches) (*Service, *string) {
	var district string
	s := New(repo, fakeTournaments{}, fakeTeams{ids: teamIDs}, fakeVenues{dzongkhag: &district}, matches)
	s.now = func() time.Time { return time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) }
	s.newID = func() (string, error) { return "event-1", nil }
	return s, &district
}

func TestGenerateCreatesFixturesMatchesAndEvent(t *testing.T) {
	repo, matches := &fakeRepo{linked: map[string]string{}}, &fakeMatches{}
	s, district := newService(repo, []string{"a", "b", "c", "d"}, matches)
	fixtures, created, err := s.Generate(context.Background(), admin, tournament, "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if created != 6 || len(fixtures) != 6 || matches.created != 6 || len(repo.linked) != 6 {
		t.Fatalf("created=%d fixtures=%d matches=%d linked=%d", created, len(fixtures), matches.created, len(repo.linked))
	}
	if *district != "Thimphu" || fixtures[0].VenueName != "Changlimithang" || fixtures[0].MatchID == "" {
		t.Fatalf("venue lookup or link wrong: %q %+v", *district, fixtures[0])
	}
	var event domain.FixturesGenerated
	if err := json.Unmarshal(repo.event.Payload, &event); err != nil || event.Fixtures != 6 || event.GeneratedBy != adminID || event.TournamentID != tournament {
		t.Fatalf("event %+v %v", event, err)
	}
	if !fixtures[0].Kickoff.Equal(time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC)) {
		t.Fatalf("default start not applied: %v", fixtures[0].Kickoff)
	}
}

func TestGenerateResumesAfterPartialFailure(t *testing.T) {
	repo, matches := &fakeRepo{linked: map[string]string{}}, &fakeMatches{failAt: 3}
	s, _ := newService(repo, []string{"a", "b", "c", "d"}, matches)
	if _, _, err := s.Generate(context.Background(), admin, tournament, "", time.Time{}); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("expected dependency failure, got %v", err)
	}
	if len(repo.linked) != 2 {
		t.Fatalf("two matches should be linked before the failure, got %d", len(repo.linked))
	}
	matches.failAt = 0
	fixtures, created, err := s.Generate(context.Background(), admin, tournament, "", time.Time{})
	if err != nil || created != 0 || len(repo.linked) != 6 {
		t.Fatalf("resume: err=%v created=%d linked=%d", err, created, len(repo.linked))
	}
	for _, f := range fixtures {
		if f.MatchID == "" {
			t.Fatal("unlinked fixture after resume")
		}
	}
}

func TestGenerateRejections(t *testing.T) {
	repo := &fakeRepo{linked: map[string]string{}}
	s, _ := newService(repo, []string{"a", "b"}, &fakeMatches{})
	if _, _, err := s.Generate(context.Background(), Caller{Role: core.RoleViewer}, tournament, "", time.Time{}); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("viewer generated fixtures")
	}
	if _, _, err := s.Generate(context.Background(), admin, "x", "", time.Time{}); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if _, _, err := s.Generate(context.Background(), admin, tournament, "knockout", time.Time{}); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("unsupported format accepted")
	}
	lonely, _ := newService(&fakeRepo{linked: map[string]string{}}, []string{"a"}, &fakeMatches{})
	if _, _, err := lonely.Generate(context.Background(), admin, tournament, "", time.Time{}); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("one team: %v", err)
	}
	var district string
	missing := New(repo, fakeTournaments{err: core.NotFound("tournament")}, fakeTeams{}, fakeVenues{dzongkhag: &district}, &fakeMatches{})
	if _, _, err := missing.Generate(context.Background(), admin, tournament, "", time.Time{}); !errors.Is(err, core.ErrNotFound) {
		t.Fatal("missing tournament not reported")
	}
	teamsDown := New(repo, fakeTournaments{}, fakeTeams{err: core.Dependency("down")}, fakeVenues{dzongkhag: &district}, &fakeMatches{})
	if _, _, err := teamsDown.Generate(context.Background(), admin, tournament, "", time.Time{}); !errors.Is(err, core.ErrDependency) {
		t.Fatal("team outage not reported")
	}
	linkFails, _ := newService(&fakeRepo{linked: map[string]string{}, linkErr: errors.New("db")}, []string{"a", "b"}, &fakeMatches{})
	if _, _, err := linkFails.Generate(context.Background(), admin, tournament, "", time.Time{}); err == nil {
		t.Fatal("link failure ignored")
	}
	badID, _ := newService(&fakeRepo{linked: map[string]string{}}, []string{"a", "b"}, &fakeMatches{})
	badID.newID = func() (string, error) { return "", errors.New("entropy") }
	if _, _, err := badID.Generate(context.Background(), admin, tournament, "", time.Time{}); err == nil {
		t.Fatal("event id failure ignored")
	}
}

func TestList(t *testing.T) {
	s, _ := newService(&fakeRepo{stored: []domain.Fixture{{ID: "f1"}}}, nil, &fakeMatches{})
	if _, _, err := s.List(context.Background(), "x", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if items, _, err := s.List(context.Background(), tournament, 10, 0); err != nil || len(items) != 1 {
		t.Fatalf("list: %v", err)
	}
}
