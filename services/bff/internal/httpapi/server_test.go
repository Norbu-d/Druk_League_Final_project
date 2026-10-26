package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	officialv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var secret = []byte("test-secret")

// calls records which RPCs ran and the bearer token each one would forward.
type calls struct {
	names   []string
	bearers []string
}

func (c *calls) add(ctx context.Context, name string) {
	c.names = append(c.names, name)
	c.bearers = append(c.bearers, servicekit.BearerFromIncoming(ctx))
}

func (c *calls) has(name string) bool {
	for _, n := range c.names {
		if n == name {
			return true
		}
	}
	return false
}

type fakeAuth struct {
	authv1.AuthServiceClient
	log *calls
}

func (f fakeAuth) Login(ctx context.Context, in *authv1.LoginRequest, _ ...grpc.CallOption) (*authv1.LoginResponse, error) {
	f.log.add(ctx, "Login")
	if in.GetPassword() != "right" {
		return nil, status.Error(codes.Unauthenticated, "authentication required: invalid username or password")
	}
	return &authv1.LoginResponse{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: timestamppb.New(time.Unix(1900000000, 0)), User: &v1.User{Id: "u1", Username: in.GetUsername(), Role: "super_admin"}}, nil
}
func (f fakeAuth) RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest, _ ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
	f.log.add(ctx, "RefreshToken")
	if in.GetRefreshToken() != "refresh" {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return &authv1.RefreshTokenResponse{AccessToken: "access2", RefreshToken: "refresh2"}, nil
}
func (f fakeAuth) RevokeSession(ctx context.Context, _ *authv1.RevokeSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeSessionResponse, error) {
	f.log.add(ctx, "RevokeSession")
	return &authv1.RevokeSessionResponse{Revoked: true}, nil
}

type fakeUsers struct {
	v1.UserServiceClient
	log *calls
}

func (f fakeUsers) ListUsers(ctx context.Context, in *v1.ListUsersRequest, _ ...grpc.CallOption) (*v1.ListUsersResponse, error) {
	f.log.add(ctx, "ListUsers:"+in.GetRole())
	return &v1.ListUsersResponse{Users: []*v1.User{{Id: "o1", Username: "official", Role: "match_official"}}}, nil
}
func (f fakeUsers) CreateUser(ctx context.Context, in *v1.CreateUserRequest, _ ...grpc.CallOption) (*v1.CreateUserResponse, error) {
	f.log.add(ctx, "CreateUser")
	if len(in.GetPassword()) < 10 {
		return nil, status.Error(codes.InvalidArgument, "invalid request: password must be at least 10 characters")
	}
	return &v1.CreateUserResponse{User: &v1.User{Id: "n1", Username: in.GetUsername(), Role: in.GetRole()}}, nil
}

type fakeTournaments struct {
	v1.TournamentServiceClient
	log *calls
}

// ListTournaments returns two pages, to prove the gateway follows page tokens.
func (f fakeTournaments) ListTournaments(ctx context.Context, in *v1.ListTournamentsRequest, _ ...grpc.CallOption) (*v1.ListTournamentsResponse, error) {
	f.log.add(ctx, "ListTournaments")
	if in.GetPageToken() == "" {
		return &v1.ListTournamentsResponse{Tournaments: []*v1.Tournament{{Id: "t1", Name: "League", Status: "registration_open"}}, NextPageToken: "1"}, nil
	}
	return &v1.ListTournamentsResponse{Tournaments: []*v1.Tournament{{Id: "t2", Name: "Cup", StartsAt: timestamppb.New(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))}}}, nil
}
func (f fakeTournaments) GetTournament(ctx context.Context, in *v1.GetTournamentRequest, _ ...grpc.CallOption) (*v1.GetTournamentResponse, error) {
	f.log.add(ctx, "GetTournament:"+in.GetId())
	switch in.GetId() {
	case "future":
		return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: "future", StartsAt: timestamppb.New(time.Date(2099, 3, 4, 0, 0, 0, 0, time.UTC))}}, nil
	case "past":
		return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: "past", StartsAt: timestamppb.New(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))}}, nil
	case "missing":
		return nil, status.Error(codes.NotFound, "not found: tournament not found")
	}
	return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: in.GetId()}}, nil
}
func (f fakeTournaments) CreateTournament(ctx context.Context, in *v1.CreateTournamentRequest, _ ...grpc.CallOption) (*v1.CreateTournamentResponse, error) {
	f.log.add(ctx, "CreateTournament")
	return &v1.CreateTournamentResponse{Tournament: &v1.Tournament{Id: "t9", Name: in.GetName(), StartsAt: in.GetStartsAt(), Status: "registration_open"}}, nil
}

type fakeTeams struct {
	v1.TeamServiceClient
	log *calls
}

func (f fakeTeams) ListTeams(ctx context.Context, in *v1.ListTeamsRequest, _ ...grpc.CallOption) (*v1.ListTeamsResponse, error) {
	f.log.add(ctx, "ListTeams:"+in.GetTournamentId())
	return &v1.ListTeamsResponse{Teams: []*v1.Team{{Id: "h", Name: "Paro FC", ShortCode: "PFC"}}}, nil
}
func (f fakeTeams) CreateTeam(ctx context.Context, in *v1.CreateTeamRequest, _ ...grpc.CallOption) (*v1.CreateTeamResponse, error) {
	f.log.add(ctx, "CreateTeam:"+in.GetShortCode())
	if in.GetShortCode() == "DUP" {
		return nil, status.Error(codes.AlreadyExists, "conflict: team short code already exists for this tournament")
	}
	return &v1.CreateTeamResponse{Team: &v1.Team{Id: "new", Name: in.GetName(), ShortCode: in.GetShortCode()}}, nil
}
func (f fakeTeams) GetTeam(ctx context.Context, in *v1.GetTeamRequest, _ ...grpc.CallOption) (*v1.GetTeamResponse, error) {
	f.log.add(ctx, "GetTeam:"+in.GetId())
	if in.GetId() == "gone" {
		return nil, status.Error(codes.Unavailable, "down")
	}
	return &v1.GetTeamResponse{Team: &v1.Team{Id: in.GetId(), Name: "Team " + in.GetId()}}, nil
}

type fakeMatches struct {
	v1.MatchServiceClient
	log *calls
}

func (f fakeMatches) ListMatches(ctx context.Context, in *v1.ListMatchesRequest, _ ...grpc.CallOption) (*v1.ListMatchesResponse, error) {
	f.log.add(ctx, "ListMatches:"+in.GetTournamentId())
	at := func(day int) *timestamppb.Timestamp {
		return timestamppb.New(time.Date(2026, 11, day, 0, 0, 0, 0, time.UTC))
	}
	return &v1.ListMatchesResponse{Matches: []*v1.Match{
		{Id: "no-kickoff", HomeTeamId: "h", AwayTeamId: "a", Status: "scheduled"},
		{Id: "final", HomeTeamId: "h", AwayTeamId: "a", Status: "completed", Kickoff: at(3)},
		{Id: "live", HomeTeamId: "h", AwayTeamId: "gone", Status: "scheduled", Kickoff: at(2), Venue: "Changlimithang"},
		{Id: "fresh", HomeTeamId: "a", AwayTeamId: "h", Status: "scheduled", Kickoff: at(1)},
		{Id: "broken", HomeTeamId: "a", AwayTeamId: "h", Status: "scheduled", Kickoff: at(4)},
	}}, nil
}
func (f fakeMatches) CreateMatch(ctx context.Context, in *v1.CreateMatchRequest, _ ...grpc.CallOption) (*v1.CreateMatchResponse, error) {
	f.log.add(ctx, "CreateMatch")
	return &v1.CreateMatchResponse{Match: &v1.Match{Id: "m9", Status: "scheduled"}}, nil
}

type fakeScores struct {
	v1.ScoreServiceClient
	log *calls
}

func (f fakeScores) GetScore(ctx context.Context, in *v1.GetScoreRequest, _ ...grpc.CallOption) (*v1.GetScoreResponse, error) {
	switch in.GetMatchId() {
	case "final":
		return &v1.GetScoreResponse{Score: &v1.Score{MatchId: "final", HomeScore: 3, AwayScore: 1, Finalized: true, UpdatedAt: timestamppb.Now()}}, nil
	case "live":
		return &v1.GetScoreResponse{Score: &v1.Score{MatchId: "live", HomeScore: 1, AwayScore: 0, UpdatedAt: timestamppb.Now()}}, nil
	case "broken":
		return nil, status.Error(codes.Unavailable, "down")
	}
	return &v1.GetScoreResponse{Score: &v1.Score{MatchId: in.GetMatchId()}}, nil
}
func (f fakeScores) UpdateLiveScore(ctx context.Context, in *v1.UpdateLiveScoreRequest, _ ...grpc.CallOption) (*v1.UpdateLiveScoreResponse, error) {
	f.log.add(ctx, "UpdateLiveScore")
	if in.GetHomeScore() > 99 {
		return nil, status.Error(codes.InvalidArgument, "invalid request: scores must be 0 to 99")
	}
	return &v1.UpdateLiveScoreResponse{Score: &v1.Score{MatchId: in.GetMatchId(), HomeScore: in.GetHomeScore(), AwayScore: in.GetAwayScore()}}, nil
}
func (f fakeScores) FinalizeMatch(ctx context.Context, in *v1.FinalizeMatchRequest, _ ...grpc.CallOption) (*v1.FinalizeMatchResponse, error) {
	f.log.add(ctx, "FinalizeMatch")
	if in.GetMatchId() == "done" {
		return nil, status.Error(codes.AlreadyExists, "conflict: match is already finalised")
	}
	return &v1.FinalizeMatchResponse{Score: &v1.Score{MatchId: in.GetMatchId(), HomeScore: 2, AwayScore: 2, Finalized: true}}, nil
}

type fakeStandings struct {
	standingsv1.StandingsServiceClient
	log *calls
}

func (f fakeStandings) GetStandings(ctx context.Context, in *standingsv1.GetStandingsRequest, _ ...grpc.CallOption) (*standingsv1.GetStandingsResponse, error) {
	f.log.add(ctx, "GetStandings:"+in.GetTournamentId())
	return &standingsv1.GetStandingsResponse{Standings: []*standingsv1.Standing{{TeamId: "h", TeamName: "Paro FC", Played: 1, Won: 1, GoalsFor: 3, GoalsAgainst: 1, GoalDifference: 2, Points: 3}}}, nil
}

type fakeFixtures struct {
	fixturev1.FixtureServiceClient
	log *calls
}

func (f fakeFixtures) GenerateFixtures(ctx context.Context, in *fixturev1.GenerateFixturesRequest, _ ...grpc.CallOption) (*fixturev1.GenerateFixturesResponse, error) {
	start := "default"
	if in.GetStartsAt() != nil {
		start = in.GetStartsAt().AsTime().Format(time.RFC3339)
	}
	f.log.add(ctx, "GenerateFixtures:"+in.GetTournamentId()+":"+start)
	return &fixturev1.GenerateFixturesResponse{CreatedCount: 6}, nil
}

type fakeOfficials struct {
	officialv1.OfficialServiceClient
	log *calls
}

func (f fakeOfficials) AssignOfficial(ctx context.Context, in *officialv1.AssignOfficialRequest, _ ...grpc.CallOption) (*officialv1.AssignOfficialResponse, error) {
	f.log.add(ctx, "AssignOfficial:"+in.GetMatchId()+":"+in.GetOfficialUserId())
	if in.GetOfficialUserId() == "viewer" {
		return nil, status.Error(codes.InvalidArgument, "invalid request: user is not a match official")
	}
	return &officialv1.AssignOfficialResponse{Assignment: &officialv1.OfficialAssignment{MatchId: in.GetMatchId()}}, nil
}

func newServer(t *testing.T) (http.Handler, *calls, string) {
	t.Helper()
	log := &calls{}
	clients := Clients{
		Auth: fakeAuth{log: log}, Users: fakeUsers{log: log}, Tournaments: fakeTournaments{log: log}, Teams: fakeTeams{log: log},
		Matches: fakeMatches{log: log}, Scores: fakeScores{log: log}, Standings: fakeStandings{log: log}, Fixtures: fakeFixtures{log: log}, Officials: fakeOfficials{log: log},
	}
	token, _, err := servicekit.SignToken(secret, servicekit.Identity{ID: "u1", Username: "admin", Role: "super_admin"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return New(clients, secret, "").Handler(), log, token
}

func do(t *testing.T, h http.Handler, method, path, token, body string) (int, any) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func field(v any, key string) any { return v.(map[string]any)[key] }

func TestHealthAndCORS(t *testing.T) {
	h, _, _ := newServer(t)
	if code, body := do(t, h, "GET", "/healthz", "", ""); code != 200 || field(body, "status") != "ok" {
		t.Fatalf("healthz: %d %v", code, body)
	}
	req := httptest.NewRequest("OPTIONS", "/api/teams", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "*" || rec.Header().Get("X-Request-ID") != "" {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("responses should carry a request id")
	}
}

func TestAuthentication(t *testing.T) {
	h, log, token := newServer(t)
	if code, _ := do(t, h, "GET", "/api/teams", "", ""); code != 401 {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := do(t, h, "GET", "/api/teams", "not-a-jwt", ""); code != 401 {
		t.Fatalf("bad token: %d", code)
	}
	if len(log.names) != 0 {
		t.Fatal("unauthenticated requests must not reach services")
	}
	if code, _ := do(t, h, "GET", "/api/teams", token, ""); code != 200 || log.bearers[0] != token {
		t.Fatalf("token should be forwarded: %d %v", code, log.bearers)
	}
	if code, _ := do(t, h, "PUT", "/api/teams", token, ""); code != 405 {
		t.Fatalf("wrong method: %d", code)
	}
}

func TestLoginRefreshLogout(t *testing.T) {
	h, log, token := newServer(t)
	code, body := do(t, h, "POST", "/api/auth/login", "", `{"username":" admin ","password":"right"}`)
	if code != 200 || field(body, "token") != "access" || field(body, "refresh_token") != "refresh" {
		t.Fatalf("login: %d %v", code, body)
	}
	if user := field(body, "user").(map[string]any); user["username"] != "admin" || user["role"] != "super_admin" || user["sub"] != "u1" {
		t.Fatalf("login user: %v", user)
	}
	if code, body := do(t, h, "POST", "/api/auth/login", "", `{"username":"admin","password":"wrong"}`); code != 401 || field(body, "error") != "Invalid username or password" {
		t.Fatalf("bad login: %d %v", code, body)
	}
	if code, _ := do(t, h, "POST", "/api/auth/login", "", `{"user":"admin"}`); code != 400 {
		t.Fatalf("unknown field: %d", code)
	}
	if code, body := do(t, h, "POST", "/api/auth/refresh", "", `{"refresh_token":"refresh"}`); code != 200 || field(body, "token") != "access2" || field(body, "refresh_token") != "refresh2" {
		t.Fatalf("refresh: %d %v", code, body)
	}
	if code, _ := do(t, h, "POST", "/api/auth/refresh", "", `{"refresh_token":"old"}`); code != 401 {
		t.Fatalf("stale refresh: %d", code)
	}
	if code, _ := do(t, h, "POST", "/api/auth/logout", token, ""); code != 200 || !log.has("RevokeSession") {
		t.Fatalf("logout: %d", code)
	}
}

func TestTournamentsAndFixtures(t *testing.T) {
	h, log, token := newServer(t)
	code, body := do(t, h, "GET", "/api/tournaments", token, "")
	list := body.([]any)
	if code != 200 || len(list) != 2 || field(list[1], "starts_at") != "2026-11-01T00:00:00Z" || field(list[0], "starts_at") != nil {
		t.Fatalf("list should follow pages: %d %v", code, body)
	}
	code, body = do(t, h, "POST", "/api/tournaments", token, `{"name":"Cup","sport":"football","season":"2026","starts_at":"2026-11-05","dzongkhag":"Paro","venue":""}`)
	if code != 201 || field(body, "id") != "t9" || field(body, "starts_at") != "2026-11-05T00:00:00Z" {
		t.Fatalf("create: %d %v", code, body)
	}
	if code, _ := do(t, h, "POST", "/api/tournaments", token, `{"name":"Cup","starts_at":"next week"}`); code != 400 {
		t.Fatalf("bad date: %d", code)
	}
	if code, body := do(t, h, "POST", "/api/tournaments/t1/fixtures", token, ""); code != 201 || field(body, "fixtures_created") != float64(6) || !log.has("GenerateFixtures:t1:default") {
		t.Fatalf("fixtures without a start date: %d %v %v", code, body, log.names)
	}
	if code, _ := do(t, h, "POST", "/api/tournaments/future/fixtures", token, ""); code != 201 || !log.has("GenerateFixtures:future:2099-03-04T04:00:00Z") {
		t.Fatalf("fixtures should start on the tournament's date at 10:00 BTT: %v", log.names)
	}
	if code, _ := do(t, h, "POST", "/api/tournaments/past/fixtures", token, ""); code != 201 || !log.has("GenerateFixtures:past:default") {
		t.Fatalf("a past start date should fall back to the default: %v", log.names)
	}
	if code, _ := do(t, h, "POST", "/api/tournaments/missing/fixtures", token, ""); code != 404 {
		t.Fatalf("missing tournament: %d", code)
	}
}

func TestTeams(t *testing.T) {
	h, log, token := newServer(t)
	if code, body := do(t, h, "GET", "/api/teams?tournament_id=t1", token, ""); code != 200 || field(body.([]any)[0], "short_code") != "PFC" || !log.has("ListTeams:t1") {
		t.Fatalf("list: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/teams", token, `{"name":"Paro","short_code":"P","dzongkhag":"Paro"}`); code != 201 || field(body, "short_code") != "PAR" {
		t.Fatalf("one-letter code should be widened: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/teams", token, `{"name":"Dup","short_code":"DUP","dzongkhag":"Paro"}`); code != 409 || field(body, "error") != "Team short code already exists for this tournament" {
		t.Fatalf("conflict: %d %v", code, body)
	}
}

func TestShortCode(t *testing.T) {
	for _, c := range []struct{ given, name, want string }{
		{"pfc", "Paro FC", "PFC"},
		{"", "Paro Football Club", "PFC"},
		{"P", "Paro", "PAR"},
		{"", "Ha", "HA"},
		{"-", "Thimphu City FC United Juniors", "TCFU"},
	} {
		if got := shortCode(c.given, c.name); got != c.want {
			t.Fatalf("shortCode(%q,%q)=%q want %q", c.given, c.name, got, c.want)
		}
	}
}

func TestListMatchesJoinsTeamsAndScores(t *testing.T) {
	h, log, token := newServer(t)
	code, body := do(t, h, "GET", "/api/matches?tournament_id=t1", token, "")
	rows := body.([]any)
	if code != 200 || len(rows) != 5 || !log.has("ListMatches:t1") {
		t.Fatalf("list: %d %v", code, body)
	}
	order := []string{}
	byID := map[string]map[string]any{}
	for _, r := range rows {
		row := r.(map[string]any)
		order = append(order, row["id"].(string))
		byID[row["id"].(string)] = row
	}
	if strings.Join(order, ",") != "fresh,live,final,broken,no-kickoff" {
		t.Fatalf("rows should be sorted by kickoff, unscheduled last: %v", order)
	}
	if r := byID["final"]; r["status"] != "completed" || r["result_home"] != float64(3) || r["live_home"] != nil || r["home"] != "Team h" {
		t.Fatalf("final row: %v", r)
	}
	if r := byID["live"]; r["status"] != "live" || r["live_home"] != float64(1) || r["away"] != "Unknown team" || r["venue"] != "Changlimithang" {
		t.Fatalf("live row (team lookup down): %v", r)
	}
	if r := byID["fresh"]; r["status"] != "scheduled" || r["live_home"] != nil || r["result_home"] != nil {
		t.Fatalf("no score yet: %v", r)
	}
	if r := byID["broken"]; r["status"] != "scheduled" || r["result_home"] != nil {
		t.Fatalf("score outage should degrade the row, not fail the list: %v", r)
	}
	lookups := 0
	for _, name := range log.names {
		if name == "GetTeam:h" {
			lookups++
		}
	}
	if lookups != 1 {
		t.Fatalf("team names should be looked up once per request, got %d", lookups)
	}
}

func TestCreateMatchAndAssignOfficial(t *testing.T) {
	h, log, token := newServer(t)
	code, body := do(t, h, "POST", "/api/matches", token, `{"tournament_id":"t1","home_team_id":"h","away_team_id":"a","venue":"","kickoff":"2026-11-01T10:00:00Z","official_id":"o1"}`)
	if code != 201 || field(body, "id") != "m9" || !log.has("AssignOfficial:m9:o1") {
		t.Fatalf("create with official: %d %v", code, body)
	}
	if code, _ := do(t, h, "POST", "/api/matches", token, `{"tournament_id":"t1","kickoff":"tomorrow"}`); code != 400 {
		t.Fatalf("bad kickoff: %d", code)
	}
	if code, _ := do(t, h, "POST", "/api/matches", token, `{"tournament_id":"t1","home_team_id":"h","away_team_id":"a","kickoff":"2026-11-01T10:00:00Z","official_id":"viewer"}`); code != 400 {
		t.Fatalf("bad official: %d", code)
	}
	if code, body := do(t, h, "PATCH", "/api/matches/m1/official", token, `{"official_id":"o1"}`); code != 200 || field(body, "match_id") != "m1" || !log.has("AssignOfficial:m1:o1") {
		t.Fatalf("assign: %d %v", code, body)
	}
}

func TestScoring(t *testing.T) {
	h, log, token := newServer(t)
	if code, body := do(t, h, "PATCH", "/api/matches/m1/live", token, `{"home":2,"away":1}`); code != 200 || field(body, "live_home") != float64(2) || field(body, "status") != "live" {
		t.Fatalf("live: %d %v", code, body)
	}
	if code, _ := do(t, h, "PATCH", "/api/matches/m1/live", token, `{"home":2}`); code != 400 {
		t.Fatalf("half score: %d", code)
	}
	if code, body := do(t, h, "PATCH", "/api/matches/m1/live", token, `{"home":100,"away":0}`); code != 400 || field(body, "error") != "Scores must be 0 to 99" {
		t.Fatalf("out of range: %d %v", code, body)
	}

	log.names = nil
	if code, body := do(t, h, "POST", "/api/matches/m1/finalize", token, `{"home":2,"away":2}`); code != 200 || field(body, "status") != "completed" || field(body, "home") != float64(2) {
		t.Fatalf("finalize: %d %v", code, body)
	}
	if strings.Join(log.names, ",") != "UpdateLiveScore,FinalizeMatch" {
		t.Fatalf("submitted score must be saved before finalising: %v", log.names)
	}
	log.names = nil
	if code, _ := do(t, h, "POST", "/api/matches/m1/finalize", token, ""); code != 200 || strings.Join(log.names, ",") != "FinalizeMatch" {
		t.Fatalf("finalize without body: %d %v", code, log.names)
	}
	if code, _ := do(t, h, "POST", "/api/matches/m1/finalize", token, `{"home":1}`); code != 400 {
		t.Fatalf("half body: %d", code)
	}
	if code, _ := do(t, h, "POST", "/api/matches/m1/finalize", token, `{"home":100,"away":0}`); code != 400 {
		t.Fatalf("invalid submitted score: %d", code)
	}
	if code, _ := do(t, h, "POST", "/api/matches/done/finalize", token, ""); code != 409 {
		t.Fatalf("already final: %d", code)
	}
}

func TestStandingsAndUsers(t *testing.T) {
	h, _, token := newServer(t)
	if code, _ := do(t, h, "GET", "/api/standings", token, ""); code != 400 {
		t.Fatalf("missing tournament: %d", code)
	}
	code, body := do(t, h, "GET", "/api/standings?tournament_id=t1", token, "")
	row := body.([]any)[0]
	if code != 200 || field(row, "team") != "Paro FC" || field(row, "points") != float64(3) || field(row, "goal_difference") != float64(2) {
		t.Fatalf("standings: %d %v", code, body)
	}
	if code, body := do(t, h, "GET", "/api/users?role=match_official", token, ""); code != 200 || field(body.([]any)[0], "role") != "match_official" {
		t.Fatalf("users: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/users", token, `{"username":"ref","password":"long-enough-pass","role":"match_official"}`); code != 201 || field(body, "id") != "n1" {
		t.Fatalf("create user: %d %v", code, body)
	}
	if code, _ := do(t, h, "POST", "/api/users", token, `{"username":"ref","password":"short","role":"viewer"}`); code != 400 {
		t.Fatalf("short password: %d", code)
	}
}

func TestErrorMapping(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int
		msg  string
	}{
		{status.Error(codes.PermissionDenied, "permission denied: only the tournament admin can create tournaments"), 403, "Only the tournament admin can create tournaments"},
		{status.Error(codes.NotFound, "not found: team not found"), 404, "Team not found"},
		{status.Error(codes.FailedPrecondition, "failed precondition: tournament is not open for registration"), 409, "Tournament is not open for registration"},
		{status.Error(codes.Unavailable, ""), 503, "Request failed"},
		{status.Error(codes.DeadlineExceeded, "slow"), 504, "Slow"},
		{status.Error(codes.Internal, "pq: secret detail"), 500, "The request could not be completed"},
		{errors.New("not a status"), 500, "The request could not be completed"},
	} {
		rec := httptest.NewRecorder()
		writeStatus(rec, c.err)
		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.code || body["error"] != c.msg {
			t.Fatalf("%v: got %d %q", c.err, rec.Code, body["error"])
		}
	}
}

func TestCollectStopsAtMaxPages(t *testing.T) {
	calls := 0
	items, err := collect(context.Background(), func(context.Context, string) ([]int, string, error) {
		calls++
		return []int{calls}, "again", nil
	})
	if err != nil || calls != maxPages || len(items) != maxPages {
		t.Fatalf("collect should stop after %d pages: %d %v", maxPages, calls, err)
	}
	if _, err := collect(context.Background(), func(context.Context, string) ([]int, string, error) { return nil, "", errors.New("boom") }); err == nil {
		t.Fatal("errors should propagate")
	}
}
