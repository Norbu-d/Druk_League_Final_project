package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

type finalizedRepo struct {
	fixture    domain.Fixture
	created    bool
	advanceErr error
	pending    []domain.Fixture
	pendingErr error
	saveErr    error
	plan       domain.KnockoutPlan
	linked     []domain.Fixture
}

func (r *finalizedRepo) SaveSchedule(context.Context, string, []domain.Fixture, domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	return nil, false, nil
}
func (r *finalizedRepo) LinkMatch(_ context.Context, fixture domain.Fixture) error {
	r.linked = append(r.linked, fixture)
	return nil
}
func (r *finalizedRepo) SaveKnockoutSchedule(_ context.Context, _ string, plan domain.KnockoutPlan, _ domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	r.plan = plan
	return plan.Fixtures, true, r.saveErr
}
func (r *finalizedRepo) AdvanceKnockout(context.Context, domain.MatchFinalized) (domain.Fixture, bool, error) {
	return r.fixture, r.created, r.advanceErr
}
func (r *finalizedRepo) PendingFixtures(context.Context, string) ([]domain.Fixture, error) {
	return r.pending, r.pendingErr
}
func (r *finalizedRepo) List(context.Context, string, int, int) ([]domain.Fixture, bool, error) {
	return nil, false, nil
}

type finalizedStandings struct {
	ids []string
	err error
}

func (s finalizedStandings) TeamIDsByRank(context.Context, string) ([]string, error) {
	return s.ids, s.err
}

func TestHandleMatchFinalizedStartsKnockoutAndBooksPendingFixtures(t *testing.T) {
	repo := &finalizedRepo{
		fixture: domain.Fixture{Format: domain.FormatLeagueKnockout},
		created: true,
		pending: []domain.Fixture{{ID: "pending", TournamentID: tournament, HomeTeamID: "a", AwayTeamID: "b"}},
	}
	matches := &fakeMatches{}
	teams := []string{
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
		"33333333-3333-4333-8333-333333333333",
		"44444444-4444-4444-8444-444444444444",
	}
	ranked := append(append([]string(nil), teams[:1]...), "55555555-5555-4555-8555-555555555555")
	ranked = append(ranked, teams[1:]...)
	service := New(repo, fakeTournaments{}, fakeTeams{ids: teams}, fakeVenues{dzongkhag: new(string)}, matches,
		finalizedStandings{ids: ranked})
	service.newID = func() (string, error) { return "event-1", nil }
	service.now = func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) }
	event := domain.MatchFinalized{TournamentID: tournament, FinalizedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}

	if err := service.HandleMatchFinalized(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.plan.Fixtures) != 2 || len(repo.linked) != 1 || repo.linked[0].MatchID == "" || matches.created != 1 {
		t.Fatalf("bracket or pending fixture not created: plan=%d linked=%+v matches=%d", len(repo.plan.Fixtures), repo.linked, matches.created)
	}
	if !repo.plan.Fixtures[0].Kickoff.Equal(event.FinalizedAt.AddDate(0, 0, 7)) {
		t.Fatalf("knockout kickoff = %v", repo.plan.Fixtures[0].Kickoff)
	}
}

func TestHandleMatchFinalizedPropagatesErrorsAndRequiresCompleteStandings(t *testing.T) {
	ctx := context.Background()
	baseEvent := domain.MatchFinalized{TournamentID: tournament}
	if err := New(&finalizedRepo{advanceErr: errors.New("advance")}, fakeTournaments{}, fakeTeams{}, fakeVenues{dzongkhag: new(string)}, &fakeMatches{}).HandleMatchFinalized(ctx, baseEvent); err == nil {
		t.Fatal("advance error ignored")
	}
	if err := New(&finalizedRepo{fixture: domain.Fixture{Format: domain.FormatLeagueKnockout}, created: true}, fakeTournaments{}, fakeTeams{}, fakeVenues{dzongkhag: new(string)}, &fakeMatches{}).HandleMatchFinalized(ctx, baseEvent); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("missing standings should fail as dependency: %v", err)
	}
	service := New(&finalizedRepo{fixture: domain.Fixture{Format: domain.FormatLeagueKnockout}, created: true}, fakeTournaments{}, fakeTeams{ids: []string{"a", "b"}}, fakeVenues{dzongkhag: new(string)}, &fakeMatches{}, finalizedStandings{ids: []string{"a"}})
	if err := service.HandleMatchFinalized(ctx, baseEvent); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("incomplete standings should fail precondition: %v", err)
	}
	if err := New(&finalizedRepo{pendingErr: errors.New("pending")}, fakeTournaments{}, fakeTeams{}, fakeVenues{dzongkhag: new(string)}, &fakeMatches{}).HandleMatchFinalized(ctx, baseEvent); err == nil {
		t.Fatal("pending fixture error ignored")
	}
}

func TestHandleMatchFinalizedPropagatesBookingAndLinkErrors(t *testing.T) {
	ctx := context.Background()
	event := domain.MatchFinalized{TournamentID: tournament}
	pending := []domain.Fixture{{ID: "pending"}}
	matches := &fakeMatches{failAt: 1}
	if err := New(&finalizedRepo{pending: pending}, fakeTournaments{}, fakeTeams{}, fakeVenues{dzongkhag: new(string)}, matches).HandleMatchFinalized(ctx, event); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("match booking error not returned: %v", err)
	}
	// A repository link error is exercised through the same service flow with a custom repo.
	repo := &linkErrorRepo{finalizedRepo: finalizedRepo{pending: pending}}
	if err := New(repo, fakeTournaments{}, fakeTeams{}, fakeVenues{dzongkhag: new(string)}, &fakeMatches{}).HandleMatchFinalized(ctx, event); err == nil {
		t.Fatal("link error ignored")
	}
}

type linkErrorRepo struct{ finalizedRepo }

func (*linkErrorRepo) LinkMatch(context.Context, domain.Fixture) error { return errors.New("link") }
