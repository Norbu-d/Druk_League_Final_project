package grpcclients

import (
	"context"
	"errors"
	"testing"

	playerv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/analytics/internal/domain"
	"google.golang.org/grpc"
)

type fakeTournamentLister struct {
	pages map[string]*v1.ListTournamentsResponse
	err   error
}

func (f fakeTournamentLister) ListTournaments(_ context.Context, request *v1.ListTournamentsRequest, _ ...grpc.CallOption) (*v1.ListTournamentsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if request.GetSeason() != "2026" {
		return nil, errors.New("season filter was not forwarded")
	}
	return f.pages[request.GetPageToken()], nil
}

type fakeTeamLister struct {
	pages map[string]*v1.ListTeamsResponse
	err   error
}

func (f fakeTeamLister) ListTeams(_ context.Context, request *v1.ListTeamsRequest, _ ...grpc.CallOption) (*v1.ListTeamsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pages[request.GetTournamentId()+"|"+request.GetPageToken()], nil
}

type fakeMatchLister struct {
	pages map[string]*v1.ListMatchesResponse
	err   error
}

func (f fakeMatchLister) ListMatches(_ context.Context, request *v1.ListMatchesRequest, _ ...grpc.CallOption) (*v1.ListMatchesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pages[request.GetTournamentId()+"|"+request.GetPageToken()], nil
}

type fakePlayerLister struct {
	pages map[string]*playerv1.ListPlayersResponse
	err   error
}

func (f fakePlayerLister) ListPlayers(_ context.Context, request *playerv1.ListPlayersRequest, _ ...grpc.CallOption) (*playerv1.ListPlayersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pages[request.GetTeamId()+"|"+request.GetPageToken()], nil
}

type fakeRegistrationLister struct {
	pages map[string]*registrationv1.ListRegistrationsResponse
	fail  string
	err   error
}

func (f fakeRegistrationLister) ListRegistrations(_ context.Context, request *registrationv1.ListRegistrationsRequest, _ ...grpc.CallOption) (*registrationv1.ListRegistrationsResponse, error) {
	if request.GetStatus() == f.fail {
		return nil, f.err
	}
	return f.pages[request.GetTournamentId()+"|"+request.GetStatus()+"|"+request.GetPageToken()], nil
}

func TestDashboardMetricsCountsFilteredPaginatedSourceRecords(t *testing.T) {
	tournaments := fakeTournamentLister{pages: map[string]*v1.ListTournamentsResponse{
		"": {Tournaments: []*v1.Tournament{
			{Id: "t1", Season: "2026", Dzongkhag: "Thimphu", Venue: "Court A"},
			{Id: "excluded", Season: "2026", Dzongkhag: "Paro", Venue: "Court C"},
		}, NextPageToken: "next"},
		"next": {Tournaments: []*v1.Tournament{{Id: "t2", Season: "2026", Dzongkhag: "thimphu", Venue: "court a"}}},
	}}
	teams := fakeTeamLister{pages: map[string]*v1.ListTeamsResponse{
		"t1|":          {Teams: []*v1.Team{{Id: "team1"}}, NextPageToken: "next-team"},
		"t1|next-team": {Teams: []*v1.Team{{Id: "team2"}}},
		"t2|":          {Teams: []*v1.Team{{Id: "team2"}}},
	}}
	matches := fakeMatchLister{pages: map[string]*v1.ListMatchesResponse{
		"t1|":           {Matches: []*v1.Match{{Id: "m1", HomeTeamId: "team1", AwayTeamId: "team4", Venue: "Court A", OfficialId: "official1", Status: "completed"}}, NextPageToken: "next-match"},
		"t1|next-match": {Matches: []*v1.Match{{Id: "m2", HomeTeamId: "team2", AwayTeamId: "team1", Venue: "Court B", OfficialId: "official1", Status: "scheduled"}}},
		"t2|":           {Matches: []*v1.Match{{Id: "m3", HomeTeamId: "team3", AwayTeamId: "team4", Venue: "court a", OfficialId: "official2", Status: "COMPLETED"}}},
	}}
	players := fakePlayerLister{pages: map[string]*playerv1.ListPlayersResponse{
		"team1|":            {Players: []*playerv1.Player{{Id: "p1"}}, NextPageToken: "next-player"},
		"team1|next-player": {Players: []*playerv1.Player{{Id: "p2"}}},
		"team2|":            {Players: []*playerv1.Player{{Id: "p3"}}},
		"team3|":            {Players: []*playerv1.Player{{Id: "p2"}}},
		"team4|":            {},
	}}
	registrations := fakeRegistrationLister{pages: map[string]*registrationv1.ListRegistrationsResponse{
		"t1|approved|":                 {Registrations: []*registrationv1.Registration{{TeamId: "team3"}}},
		"t1|pending|":                  {Registrations: []*registrationv1.Registration{{Id: "r1"}, {Id: "r2"}}, NextPageToken: "next-registration"},
		"t1|pending|next-registration": {Registrations: []*registrationv1.Registration{{Id: "r3"}}},
		"t2|approved|":                 {Registrations: []*registrationv1.Registration{{TeamId: "team1"}}},
		"t2|pending|":                  {},
	}}
	dashboard := &Dashboard{tournaments: tournaments, teams: teams, matches: matches, players: players, registrations: registrations}

	got, err := dashboard.DashboardMetrics(context.Background(), "2026", "Thimphu")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.DashboardMetrics{Tournaments: 2, Teams: 4, Players: 3, Matches: 3, CompletedMatches: 2, Venues: 2, Officials: 2, PendingRegistrations: 3}
	if got != want {
		t.Fatalf("metrics=%+v want %+v", got, want)
	}
}

func TestDashboardMetricsPropagatesUpstreamFailures(t *testing.T) {
	upstreamErr := errors.New("upstream unavailable")
	tests := []struct {
		name      string
		configure func(*Dashboard)
	}{
		{"tournaments", func(d *Dashboard) { d.tournaments = fakeTournamentLister{err: upstreamErr} }},
		{"teams", func(d *Dashboard) { d.teams = fakeTeamLister{err: upstreamErr} }},
		{"approved registrations", func(d *Dashboard) { d.registrations = fakeRegistrationLister{fail: "approved", err: upstreamErr} }},
		{"pending registrations", func(d *Dashboard) {
			d.registrations = fakeRegistrationLister{fail: "pending", err: upstreamErr, pages: map[string]*registrationv1.ListRegistrationsResponse{"t1|approved|": {}}}
		}},
		{"matches", func(d *Dashboard) { d.matches = fakeMatchLister{err: upstreamErr} }},
		{"players", func(d *Dashboard) {
			d.players = fakePlayerLister{err: upstreamErr}
			d.teams = fakeTeamLister{pages: map[string]*v1.ListTeamsResponse{"t1|": {Teams: []*v1.Team{{Id: "team1"}}}}}
			d.registrations = fakeRegistrationLister{pages: map[string]*registrationv1.ListRegistrationsResponse{"t1|approved|": {}, "t1|pending|": {}}}
			d.matches = fakeMatchLister{pages: map[string]*v1.ListMatchesResponse{"t1|": {}}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dashboard := minimalDashboard()
			test.configure(dashboard)
			if _, err := dashboard.DashboardMetrics(context.Background(), "2026", "Thimphu"); !errors.Is(err, upstreamErr) {
				t.Fatalf("error=%v want %v", err, upstreamErr)
			}
		})
	}
}

func TestCollectPagesRejectsRepeatedPaginationToken(t *testing.T) {
	calls := 0
	_, err := collectPages(func(token string) ([]int, string, error) {
		calls++
		if token == "" {
			return []int{1}, "repeat", nil
		}
		return []int{2}, "repeat", nil
	})
	if err == nil || calls != 2 {
		t.Fatalf("collectPages() calls=%d err=%v, want repeated-token error", calls, err)
	}
}

func minimalDashboard() *Dashboard {
	return &Dashboard{
		tournaments:   fakeTournamentLister{pages: map[string]*v1.ListTournamentsResponse{"": {Tournaments: []*v1.Tournament{{Id: "t1", Season: "2026", Dzongkhag: "Thimphu"}}}}},
		teams:         fakeTeamLister{pages: map[string]*v1.ListTeamsResponse{"t1|": {}}},
		matches:       fakeMatchLister{pages: map[string]*v1.ListMatchesResponse{"t1|": {}}},
		players:       fakePlayerLister{pages: map[string]*playerv1.ListPlayersResponse{}},
		registrations: fakeRegistrationLister{pages: map[string]*registrationv1.ListRegistrationsResponse{"t1|approved|": {}, "t1|pending|": {}}},
	}
}
