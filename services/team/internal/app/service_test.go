package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/team/internal/domain"
)

const (
	teamID     = "11111111-1111-4111-8111-111111111111"
	openCup    = "22222222-2222-4222-8222-222222222222"
	closedCup  = "33333333-3333-4333-8333-333333333333"
	missingCup = "44444444-4444-4444-8444-444444444444"
	managerID  = "55555555-5555-4555-8555-555555555555"
	downCup    = "66666666-6666-4666-8666-666666666666"
)

type fakeRepo struct {
	created []domain.Team
	filter  domain.Filter
}

func (f *fakeRepo) Create(_ context.Context, t domain.Team) (domain.Team, error) {
	t.ID = teamID
	f.created = append(f.created, t)
	return t, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Team, error) {
	if id != teamID {
		return domain.Team{}, core.NotFound("team not found")
	}
	return domain.Team{ID: id}, nil
}
func (f *fakeRepo) List(_ context.Context, filter domain.Filter, _, _ int) ([]domain.Team, bool, error) {
	f.filter = filter
	return []domain.Team{{ID: teamID}}, false, nil
}

type fakeTournaments struct{}

func (fakeTournaments) Get(_ context.Context, id string) (domain.TournamentRef, error) {
	switch id {
	case openCup:
		return domain.TournamentRef{ID: id, Status: domain.TournamentOpen}, nil
	case closedCup:
		return domain.TournamentRef{ID: id, Status: "in_progress"}, nil
	case downCup:
		return domain.TournamentRef{}, core.Dependency("down")
	}
	return domain.TournamentRef{}, core.NotFound("tournament not found")
}

type fakeDzongkhags struct{}

func (fakeDzongkhags) Canonical(_ context.Context, name string) (string, error) {
	if strings.EqualFold(strings.TrimSpace(name), "paro") {
		return "Paro", nil
	}
	return "", core.Invalid("unknown dzongkhag")
}

var (
	admin   = Caller{ID: "admin", Role: core.RoleSuperAdmin}
	manager = Caller{ID: managerID, Role: core.RoleTeamManager}
)

func newService() (*Service, *fakeRepo) {
	repo := &fakeRepo{}
	return New(repo, fakeTournaments{}, fakeDzongkhags{}), repo
}

func TestCreateByAdmin(t *testing.T) {
	s, repo := newService()
	team, err := s.Create(context.Background(), admin, domain.Team{Name: "Paro FC", ShortCode: "pfc", Dzongkhag: "paro", TournamentID: closedCup, ManagerID: managerID})
	if err != nil {
		t.Fatal(err)
	}
	if team.Dzongkhag != "Paro" || team.ShortCode != "PFC" || team.ManagerID != "" || len(repo.created) != 1 {
		t.Fatalf("admin team should be canonical and unmanaged: %+v", team)
	}
}

func TestCreateByManagerOwnsTeam(t *testing.T) {
	s, _ := newService()
	team, err := s.Create(context.Background(), manager, domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro", TournamentID: openCup})
	if err != nil || team.ManagerID != managerID {
		t.Fatalf("manager should own the team: %+v %v", team, err)
	}
	if team, err := s.Create(context.Background(), manager, domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro"}); err != nil || team.TournamentID != "" {
		t.Fatalf("team without tournament: %+v %v", team, err)
	}
}

func TestCreateRejections(t *testing.T) {
	valid := domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro"}
	with := func(tournament string) domain.Team { v := valid; v.TournamentID = tournament; return v }
	cases := []struct {
		name   string
		caller Caller
		team   domain.Team
		want   error
	}{
		{"viewer", Caller{ID: "v", Role: core.RoleViewer}, valid, core.ErrForbidden},
		{"official", Caller{ID: "o", Role: core.RoleMatchOfficial}, valid, core.ErrForbidden},
		{"invalid team", admin, domain.Team{Name: "P", ShortCode: "PFC", Dzongkhag: "Paro"}, core.ErrInvalid},
		{"unknown dzongkhag", admin, domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Atlantis"}, core.ErrInvalid},
		{"ghost tournament", admin, with(missingCup), core.ErrNotFound},
		{"tournament service down", admin, with(downCup), core.ErrDependency},
		{"manager into closed tournament", manager, with(closedCup), core.ErrPrecondition},
	}
	for _, c := range cases {
		s, repo := newService()
		if _, err := s.Create(context.Background(), c.caller, c.team); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
		if len(repo.created) != 0 {
			t.Fatalf("%s: rejected team was stored", c.name)
		}
	}
}

func TestGetAndList(t *testing.T) {
	s, repo := newService()
	if _, err := s.Get(context.Background(), "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if team, err := s.Get(context.Background(), teamID); err != nil || team.ID != teamID {
		t.Fatalf("get: %v", err)
	}
	if _, _, err := s.List(context.Background(), domain.Filter{TournamentID: " " + openCup, Dzongkhag: "paro", ManagerID: managerID}, 10, 0); err != nil {
		t.Fatal(err)
	}
	if repo.filter != (domain.Filter{TournamentID: openCup, Dzongkhag: "Paro", ManagerID: managerID}) {
		t.Fatalf("filter not normalised: %+v", repo.filter)
	}
	for name, f := range map[string]domain.Filter{"tournament": {TournamentID: "x"}, "manager": {ManagerID: "x"}, "dzongkhag": {Dzongkhag: "Atlantis"}} {
		if _, _, err := s.List(context.Background(), f, 10, 0); !errors.Is(err, core.ErrInvalid) {
			t.Fatalf("bad %s filter accepted: %v", name, err)
		}
	}
}
