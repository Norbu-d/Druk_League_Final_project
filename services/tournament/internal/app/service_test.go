package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/tournament/internal/domain"
)

const tournamentID = "11111111-1111-4111-8111-111111111111"

type fakeRepo struct {
	created []domain.Tournament
	season  string
}

func (f *fakeRepo) Create(_ context.Context, t domain.Tournament) (domain.Tournament, error) {
	t.ID = tournamentID
	f.created = append(f.created, t)
	return t, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Tournament, error) {
	if id != tournamentID {
		return domain.Tournament{}, core.NotFound("tournament not found")
	}
	return domain.Tournament{ID: id}, nil
}
func (f *fakeRepo) List(_ context.Context, season string, _, _ int) ([]domain.Tournament, bool, error) {
	f.season = season
	return []domain.Tournament{{ID: tournamentID}}, false, nil
}

type fakeDzongkhags struct{ err error }

func (f fakeDzongkhags) Canonical(_ context.Context, name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if strings.EqualFold(strings.TrimSpace(name), "thimphu") {
		return "Thimphu", nil
	}
	return "", core.Invalid("unknown dzongkhag")
}

func TestCreateOpensForRegistration(t *testing.T) {
	repo := &fakeRepo{}
	got, err := New(repo, fakeDzongkhags{}).Create(context.Background(), core.RoleSuperAdmin, domain.Tournament{Name: "National League", Sport: "Football", Season: "2026", Dzongkhag: "thimphu", Status: domain.StatusCompleted})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dzongkhag != "Thimphu" || got.Status != domain.StatusRegistrationOpen || got.Sport != "football" || len(repo.created) != 1 {
		t.Fatalf("unexpected tournament %+v", got)
	}
}

func TestCreateRejections(t *testing.T) {
	valid := domain.Tournament{Name: "National League", Sport: "football", Season: "2026", Dzongkhag: "Thimphu"}
	cases := []struct {
		name       string
		role       string
		tournament domain.Tournament
		dz         fakeDzongkhags
		want       error
	}{
		{"team manager", core.RoleTeamManager, valid, fakeDzongkhags{}, core.ErrForbidden},
		{"official", core.RoleMatchOfficial, valid, fakeDzongkhags{}, core.ErrForbidden},
		{"no name", core.RoleSuperAdmin, domain.Tournament{Sport: "football", Season: "2026", Dzongkhag: "Thimphu"}, fakeDzongkhags{}, core.ErrInvalid},
		{"unknown district", core.RoleSuperAdmin, domain.Tournament{Name: "National League", Sport: "football", Season: "2026", Dzongkhag: "Atlantis"}, fakeDzongkhags{}, core.ErrInvalid},
		{"directory down", core.RoleSuperAdmin, valid, fakeDzongkhags{err: core.Dependency("down")}, core.ErrDependency},
	}
	for _, c := range cases {
		repo := &fakeRepo{}
		if _, err := New(repo, c.dz).Create(context.Background(), c.role, c.tournament); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
		if len(repo.created) != 0 {
			t.Fatalf("%s: rejected tournament was stored", c.name)
		}
	}
}

func TestGetAndList(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, fakeDzongkhags{})
	if _, err := s.Get(context.Background(), "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if got, err := s.Get(context.Background(), tournamentID); err != nil || got.ID != tournamentID {
		t.Fatalf("get: %v", err)
	}
	if _, _, err := s.List(context.Background(), " 2026 ", 10, 0); err != nil || repo.season != "2026" {
		t.Fatalf("list: %v %q", err, repo.season)
	}
}
