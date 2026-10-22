package app

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/player/internal/domain"
)

const (
	playerID      = "11111111-1111-4111-8111-111111111111"
	ownTeam       = "22222222-2222-4222-8222-222222222222"
	otherTeam     = "33333333-3333-4333-8333-333333333333"
	unmanagedTeam = "44444444-4444-4444-8444-444444444444"
	missingTeam   = "55555555-5555-4555-8555-555555555555"
	managerID     = "66666666-6666-4666-8666-666666666666"
)

type fakeRepo struct {
	players map[string]domain.Player
	written []domain.Player
	teamID  string
}

func newRepo() *fakeRepo {
	return &fakeRepo{players: map[string]domain.Player{playerID: {ID: playerID, TeamID: ownTeam, GivenName: "Sonam", ShirtNumber: "9"}}}
}

func (f *fakeRepo) Create(_ context.Context, p domain.Player) (domain.Player, error) {
	p.ID = playerID
	f.written = append(f.written, p)
	return p, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Player, error) {
	p, ok := f.players[id]
	if !ok {
		return domain.Player{}, core.NotFound("player not found")
	}
	return p, nil
}
func (f *fakeRepo) List(_ context.Context, teamID string, _, _ int) ([]domain.Player, bool, error) {
	f.teamID = teamID
	return []domain.Player{{ID: playerID}}, false, nil
}
func (f *fakeRepo) Update(_ context.Context, p domain.Player) (domain.Player, error) {
	f.written = append(f.written, p)
	return p, nil
}

type fakeTeams struct{}

func (fakeTeams) Get(_ context.Context, id string) (domain.TeamRef, error) {
	switch id {
	case ownTeam:
		return domain.TeamRef{ID: id, ManagerID: managerID}, nil
	case otherTeam:
		return domain.TeamRef{ID: id, ManagerID: "77777777-7777-4777-8777-777777777777"}, nil
	case unmanagedTeam:
		return domain.TeamRef{ID: id}, nil
	}
	return domain.TeamRef{}, core.NotFound("team not found")
}

var (
	admin   = Caller{ID: "admin", Role: core.RoleSuperAdmin}
	manager = Caller{ID: managerID, Role: core.RoleTeamManager}
)

func TestCreate(t *testing.T) {
	for name, c := range map[string]struct {
		caller Caller
		team   string
	}{"admin, any team": {admin, unmanagedTeam}, "manager, own team": {manager, ownTeam}} {
		repo := newRepo()
		p, err := New(repo, fakeTeams{}).Create(context.Background(), c.caller, domain.Player{TeamID: c.team, GivenName: " Karma ", ShirtNumber: "07"})
		if err != nil || p.GivenName != "Karma" || p.ShirtNumber != "7" || len(repo.written) != 1 {
			t.Fatalf("%s: %+v %v", name, p, err)
		}
	}
}

func TestCreateRejections(t *testing.T) {
	cases := []struct {
		name   string
		caller Caller
		player domain.Player
		want   error
	}{
		{"viewer", Caller{ID: "v", Role: core.RoleViewer}, domain.Player{TeamID: ownTeam, GivenName: "Karma"}, core.ErrForbidden},
		{"official", Caller{ID: "o", Role: core.RoleMatchOfficial}, domain.Player{TeamID: ownTeam, GivenName: "Karma"}, core.ErrForbidden},
		{"invalid player", admin, domain.Player{TeamID: ownTeam}, core.ErrInvalid},
		{"missing team", admin, domain.Player{TeamID: missingTeam, GivenName: "Karma"}, core.ErrNotFound},
		{"manager, other team", manager, domain.Player{TeamID: otherTeam, GivenName: "Karma"}, core.ErrForbidden},
		{"manager, unmanaged team", manager, domain.Player{TeamID: unmanagedTeam, GivenName: "Karma"}, core.ErrForbidden},
	}
	for _, c := range cases {
		repo := newRepo()
		if _, err := New(repo, fakeTeams{}).Create(context.Background(), c.caller, c.player); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
		if len(repo.written) != 0 {
			t.Fatalf("%s: rejected player was stored", c.name)
		}
	}
}

func TestUpdateKeepsTeam(t *testing.T) {
	repo := newRepo()
	p, err := New(repo, fakeTeams{}).Update(context.Background(), manager, domain.Player{ID: playerID, TeamID: otherTeam, GivenName: "Sonam", FamilyName: "Choki", ShirtNumber: "10"})
	if err != nil || p.TeamID != ownTeam || p.FamilyName != "Choki" || p.ShirtNumber != "10" {
		t.Fatalf("update: %+v %v", p, err)
	}
}

func TestUpdateRejections(t *testing.T) {
	cases := []struct {
		name   string
		caller Caller
		player domain.Player
		want   error
	}{
		{"viewer", Caller{ID: "v", Role: core.RoleViewer}, domain.Player{ID: playerID, GivenName: "Sonam"}, core.ErrForbidden},
		{"bad id", admin, domain.Player{ID: "x", GivenName: "Sonam"}, core.ErrInvalid},
		{"missing player", admin, domain.Player{ID: missingTeam, GivenName: "Sonam"}, core.ErrNotFound},
		{"invalid change", admin, domain.Player{ID: playerID, GivenName: "Sonam", ShirtNumber: "0"}, core.ErrInvalid},
		{"other manager", Caller{ID: "77777777-7777-4777-8777-777777777777", Role: core.RoleTeamManager}, domain.Player{ID: playerID, GivenName: "Sonam"}, core.ErrForbidden},
	}
	for _, c := range cases {
		repo := newRepo()
		if _, err := New(repo, fakeTeams{}).Update(context.Background(), c.caller, c.player); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
		if len(repo.written) != 0 {
			t.Fatalf("%s: rejected update was stored", c.name)
		}
	}
}

func TestGetAndList(t *testing.T) {
	repo := newRepo()
	s := New(repo, fakeTeams{})
	if _, err := s.Get(context.Background(), "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if p, err := s.Get(context.Background(), playerID); err != nil || p.ID != playerID {
		t.Fatalf("get: %v", err)
	}
	if _, _, err := s.List(context.Background(), " "+ownTeam, 10, 0); err != nil || repo.teamID != ownTeam {
		t.Fatalf("list: %v %q", err, repo.teamID)
	}
	if _, _, err := s.List(context.Background(), "x", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad team filter accepted")
	}
}
