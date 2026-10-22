package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"github.com/drukleague/platform/services/registration/internal/ports"
)

const (
	tournamentID = "11111111-1111-4111-8111-111111111111"
	teamID       = "22222222-2222-4222-8222-222222222222"
	managerID    = "33333333-3333-4333-8333-333333333333"
	adminID      = "44444444-4444-4444-8444-444444444444"
	regID        = "55555555-5555-4555-8555-555555555555"
)

type fakeRepo struct {
	created       []domain.Registration
	createErr     error
	reviewID      string
	reviewDecide  string
	reviewReason  string
	reviewer      string
	filter        ports.Filter
	limit, offset int
}

func (f *fakeRepo) Create(_ context.Context, r domain.Registration) (domain.Registration, error) {
	if f.createErr != nil {
		return domain.Registration{}, f.createErr
	}
	r.ID, r.Status = regID, domain.StatusPending
	f.created = append(f.created, r)
	return r, nil
}

func (f *fakeRepo) Review(_ context.Context, id, decision, reason, reviewer string) (domain.Registration, error) {
	f.reviewID, f.reviewDecide, f.reviewReason, f.reviewer = id, decision, reason, reviewer
	return domain.Registration{ID: id, Status: decision, ReviewReason: reason, ReviewedBy: reviewer}, nil
}

func (f *fakeRepo) List(_ context.Context, filter ports.Filter, limit, offset int) ([]domain.Registration, bool, error) {
	f.filter, f.limit, f.offset = filter, limit, offset
	return []domain.Registration{{ID: regID}}, true, nil
}

type fakeTeams struct {
	team domain.TeamRef
	err  error
}

func (f fakeTeams) Get(context.Context, string) (domain.TeamRef, error) { return f.team, f.err }

type fakeTournaments struct {
	tournament domain.TournamentRef
	err        error
}

func (f fakeTournaments) Get(context.Context, string) (domain.TournamentRef, error) {
	return f.tournament, f.err
}

var (
	admin      = Caller{ID: adminID, Role: core.RoleSuperAdmin}
	manager    = Caller{ID: managerID, Role: core.RoleTeamManager}
	ownedTeam  = fakeTeams{team: domain.TeamRef{ID: teamID, ManagerID: managerID}}
	openTourn  = fakeTournaments{tournament: domain.TournamentRef{ID: tournamentID, Status: domain.TournamentOpen}}
	closedTour = fakeTournaments{tournament: domain.TournamentRef{ID: tournamentID, Status: "in_progress"}}
)

func TestRegisterByAdminAndManager(t *testing.T) {
	for _, c := range []struct {
		name   string
		caller Caller
		teams  fakeTeams
	}{
		{"admin registers any team", admin, fakeTeams{team: domain.TeamRef{ID: teamID}}},
		{"manager registers own team", manager, ownedTeam},
	} {
		repo := &fakeRepo{}
		got, err := New(repo, c.teams, openTourn).Register(context.Background(), c.caller, " "+tournamentID+" ", teamID)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.TournamentID != tournamentID || got.TeamID != teamID || got.RegisteredBy != c.caller.ID || got.Status != domain.StatusPending {
			t.Fatalf("%s: unexpected registration %+v", c.name, got)
		}
	}
}

func TestRegisterRejections(t *testing.T) {
	cases := []struct {
		name        string
		caller      Caller
		tournament  string
		team        string
		teams       fakeTeams
		tournaments fakeTournaments
		want        error
	}{
		{"viewer cannot register", Caller{ID: "v", Role: core.RoleViewer}, tournamentID, teamID, ownedTeam, openTourn, core.ErrForbidden},
		{"official cannot register", Caller{ID: "o", Role: core.RoleMatchOfficial}, tournamentID, teamID, ownedTeam, openTourn, core.ErrForbidden},
		{"bad tournament id", admin, "x", teamID, ownedTeam, openTourn, core.ErrInvalid},
		{"bad team id", admin, tournamentID, "x", ownedTeam, openTourn, core.ErrInvalid},
		{"team missing", admin, tournamentID, teamID, fakeTeams{err: core.NotFound("team not found")}, openTourn, core.ErrNotFound},
		{"tournament missing", admin, tournamentID, teamID, ownedTeam, fakeTournaments{err: core.NotFound("tournament not found")}, core.ErrNotFound},
		{"team service down", admin, tournamentID, teamID, fakeTeams{err: core.Dependency("down")}, openTourn, core.ErrDependency},
		{"registration closed", admin, tournamentID, teamID, ownedTeam, closedTour, core.ErrPrecondition},
		{"manager of another team", Caller{ID: adminID, Role: core.RoleTeamManager}, tournamentID, teamID, ownedTeam, openTourn, core.ErrForbidden},
		{"team has no manager", manager, tournamentID, teamID, fakeTeams{team: domain.TeamRef{ID: teamID}}, openTourn, core.ErrForbidden},
	}
	for _, c := range cases {
		repo := &fakeRepo{}
		if _, err := New(repo, c.teams, c.tournaments).Register(context.Background(), c.caller, c.tournament, c.team); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, err, c.want)
		}
		if len(repo.created) != 0 {
			t.Fatalf("%s: nothing should be stored", c.name)
		}
	}
}

func TestRegisterDuplicateComesFromRepository(t *testing.T) {
	repo := &fakeRepo{createErr: core.Conflict("already registered")}
	if _, err := New(repo, ownedTeam, openTourn).Register(context.Background(), manager, tournamentID, teamID); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestReview(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, ownedTeam, openTourn)
	got, err := s.Review(context.Background(), admin, " "+regID+" ", true, "")
	if err != nil || got.Status != domain.StatusApproved || repo.reviewer != adminID || repo.reviewID != regID {
		t.Fatalf("approve: %+v %v", got, err)
	}
	got, err = s.Review(context.Background(), admin, regID, false, "  roster incomplete ")
	if err != nil || got.Status != domain.StatusRejected || repo.reviewReason != "roster incomplete" {
		t.Fatalf("reject: %+v %v", got, err)
	}
}

func TestReviewRejections(t *testing.T) {
	long := strings.Repeat("x", maxReasonLength+1)
	cases := []struct {
		name     string
		caller   Caller
		id       string
		approved bool
		reason   string
		want     error
	}{
		{"manager cannot review", manager, regID, true, "", core.ErrForbidden},
		{"official cannot review", Caller{ID: "o", Role: core.RoleMatchOfficial}, regID, true, "", core.ErrForbidden},
		{"bad id", admin, "x", true, "", core.ErrInvalid},
		{"rejection needs a reason", admin, regID, false, "   ", core.ErrInvalid},
		{"reason too long", admin, regID, false, long, core.ErrInvalid},
	}
	for _, c := range cases {
		repo := &fakeRepo{}
		if _, err := New(repo, ownedTeam, openTourn).Review(context.Background(), c.caller, c.id, c.approved, c.reason); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, err, c.want)
		}
		if repo.reviewID != "" {
			t.Fatalf("%s: nothing should be reviewed", c.name)
		}
	}
}

func TestListScopes(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, ownedTeam, openTourn)
	if _, more, err := s.List(context.Background(), admin, tournamentID, domain.StatusPending, 10, 20); err != nil || !more ||
		repo.filter != (ports.Filter{TournamentID: tournamentID, Status: domain.StatusPending}) || repo.limit != 10 || repo.offset != 20 {
		t.Fatalf("admin list: %v %+v", err, repo.filter)
	}
	if _, _, err := s.List(context.Background(), Caller{ID: "svc", Role: core.RoleService}, "", "", 10, 0); err != nil || repo.filter != (ports.Filter{}) {
		t.Fatalf("service list: %v %+v", err, repo.filter)
	}
	if _, _, err := s.List(context.Background(), manager, "", "", 10, 0); err != nil || repo.filter.RegisteredBy != managerID {
		t.Fatalf("manager must only see own registrations: %v %+v", err, repo.filter)
	}
	cases := map[string]struct {
		caller     Caller
		tournament string
		status     string
		want       error
	}{
		"viewer":         {Caller{ID: "v", Role: core.RoleViewer}, "", "", core.ErrForbidden},
		"official":       {Caller{ID: "o", Role: core.RoleMatchOfficial}, "", "", core.ErrForbidden},
		"bad tournament": {admin, "x", "", core.ErrInvalid},
		"bad status":     {admin, "", "done", core.ErrInvalid},
	}
	for name, c := range cases {
		if _, _, err := s.List(context.Background(), c.caller, c.tournament, c.status, 10, 0); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}
