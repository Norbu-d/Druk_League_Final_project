package httpapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	playerv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRegistrations struct {
	registrationv1.RegistrationServiceClient
	log *calls
}

func (f fakeRegistrations) RegisterTeam(ctx context.Context, in *registrationv1.RegisterTeamRequest, _ ...grpc.CallOption) (*registrationv1.RegisterTeamResponse, error) {
	f.log.add(ctx, "RegisterTeam:"+in.GetTournamentId()+":"+in.GetTeamId())
	if in.GetTournamentId() == "closed" {
		return nil, status.Error(codes.FailedPrecondition, "failed precondition: registration is not open for this tournament")
	}
	return &registrationv1.RegisterTeamResponse{Registration: &registrationv1.Registration{Id: "r1", TournamentId: in.GetTournamentId(), TeamId: in.GetTeamId(), Status: "pending"}}, nil
}
func (f fakeRegistrations) ReviewRegistration(ctx context.Context, in *registrationv1.ReviewRegistrationRequest, _ ...grpc.CallOption) (*registrationv1.ReviewRegistrationResponse, error) {
	decision := "rejected"
	if in.GetApproved() {
		decision = "approved"
	}
	f.log.add(ctx, "ReviewRegistration:"+in.GetRegistrationId()+":"+decision)
	if !in.GetApproved() && in.GetReason() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request: a reason is required when rejecting a registration")
	}
	return &registrationv1.ReviewRegistrationResponse{Registration: &registrationv1.Registration{Id: in.GetRegistrationId(), Status: decision}}, nil
}
func (f fakeRegistrations) ListRegistrations(ctx context.Context, in *registrationv1.ListRegistrationsRequest, _ ...grpc.CallOption) (*registrationv1.ListRegistrationsResponse, error) {
	f.log.add(ctx, "ListRegistrations:"+in.GetTournamentId()+":"+in.GetStatus())
	return &registrationv1.ListRegistrationsResponse{Registrations: []*registrationv1.Registration{{Id: "r1", TournamentId: "t1", TeamId: "h", Status: "pending"}}}, nil
}

type fakePlayers struct {
	playerv1.PlayerServiceClient
	log *calls
}

func (f fakePlayers) ListPlayers(ctx context.Context, in *playerv1.ListPlayersRequest, _ ...grpc.CallOption) (*playerv1.ListPlayersResponse, error) {
	f.log.add(ctx, "ListPlayers:"+in.GetTeamId())
	return &playerv1.ListPlayersResponse{Players: []*playerv1.Player{{Id: "p1", TeamId: in.GetTeamId(), GivenName: "Sonam", FamilyName: "Dorji", ShirtNumber: "9"}}}, nil
}
func (f fakePlayers) CreatePlayer(ctx context.Context, in *playerv1.CreatePlayerRequest, _ ...grpc.CallOption) (*playerv1.CreatePlayerResponse, error) {
	f.log.add(ctx, "CreatePlayer:"+in.GetTeamId())
	return &playerv1.CreatePlayerResponse{Player: &playerv1.Player{Id: "p2", TeamId: in.GetTeamId(), GivenName: in.GetGivenName(), FamilyName: in.GetFamilyName(), ShirtNumber: in.GetShirtNumber()}}, nil
}
func (f fakePlayers) UpdatePlayer(ctx context.Context, in *playerv1.UpdatePlayerRequest, _ ...grpc.CallOption) (*playerv1.UpdatePlayerResponse, error) {
	f.log.add(ctx, "UpdatePlayer:"+in.GetId())
	return &playerv1.UpdatePlayerResponse{Player: &playerv1.Player{Id: in.GetId(), GivenName: in.GetGivenName(), FamilyName: in.GetFamilyName(), ShirtNumber: in.GetShirtNumber()}}, nil
}

type fakeVenues struct {
	venuev1.VenueServiceClient
	log *calls
}

func (f fakeVenues) ListVenues(ctx context.Context, in *venuev1.ListVenuesRequest, _ ...grpc.CallOption) (*venuev1.ListVenuesResponse, error) {
	f.log.add(ctx, "ListVenues:"+in.GetDzongkhag())
	return &venuev1.ListVenuesResponse{Venues: []*venuev1.Venue{{Id: "v1", Name: "Changlimithang", Dzongkhag: "Thimphu", Capacity: 15000}}}, nil
}
func (f fakeVenues) CreateVenue(ctx context.Context, in *venuev1.CreateVenueRequest, _ ...grpc.CallOption) (*venuev1.CreateVenueResponse, error) {
	f.log.add(ctx, "CreateVenue:"+in.GetName())
	return &venuev1.CreateVenueResponse{Venue: &venuev1.Venue{Id: "v2", Name: in.GetName(), Dzongkhag: in.GetDzongkhag(), Capacity: in.GetCapacity()}}, nil
}

func roleToken(t *testing.T, id, role string) string {
	t.Helper()
	token, _, err := servicekit.SignToken(secret, servicekit.Identity{ID: id, Username: id, Role: role}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// assertServiceBearer checks that the last call carried the BFF's own service token.
func assertServiceBearer(t *testing.T, log *calls) {
	t.Helper()
	identity, err := servicekit.VerifyToken(log.bearers[len(log.bearers)-1], secret)
	if err != nil || identity.Role != servicekit.RoleService || identity.Username != "bff-service" {
		t.Fatalf("public call should use the BFF service token, got %+v %v", identity, err)
	}
}

func TestPublicRoutesUseServiceTokenAndHideAccountIDs(t *testing.T) {
	h, log, _ := newServer(t)
	code, body := do(t, h, "GET", "/api/public/teams", "", "")
	if code != 200 || field(body.([]any)[0], "name") != "Paro FC" || field(body.([]any)[0], "manager_id") != nil {
		t.Fatalf("public teams: %d %v", code, body)
	}
	assertServiceBearer(t, log)

	code, body = do(t, h, "GET", "/api/public/matches", "", "")
	if code != 200 || len(body.([]any)) != 5 {
		t.Fatalf("public matches: %d %v", code, body)
	}
	for _, row := range body.([]any) {
		if _, present := row.(map[string]any)["official_id"]; present {
			t.Fatalf("public match rows must not carry official_id: %v", row)
		}
	}
	if code, _ := do(t, h, "GET", "/api/public/tournaments", "", ""); code != 200 {
		t.Fatalf("public tournaments: %d", code)
	}
	if code, body := do(t, h, "GET", "/api/public/standings?tournament_id=t1", "", ""); code != 200 || field(body.([]any)[0], "team") != "Paro FC" {
		t.Fatalf("public standings: %d %v", code, body)
	}

	// A browser session cookie is ignored on public routes: the service token is used, never the user's.
	admin := signedTestToken("browser")
	if rec, _ := doWithCookie(h, "GET", "/api/public/teams", sessionCookieName, admin, ""); rec.Code != 200 {
		t.Fatalf("public with cookie: %d", rec.Code)
	}
	assertServiceBearer(t, log)

	// Public routes are read-only.
	if code, _ := do(t, h, "POST", "/api/public/teams", "", `{}`); code != 405 {
		t.Fatalf("public write must be refused: %d", code)
	}
}

func TestSignupCreatesSignedInViewer(t *testing.T) {
	h, log, _ := newServer(t)
	rec, body := doWithCookie(h, "POST", "/api/auth/signup", "", "", `{"username":" fan ","password":"right-and-long"}`)
	if rec.Code != 201 || field(body, "signed_in") != true || field(field(body, "user"), "role") != "viewer" {
		t.Fatalf("signup: %d %v", rec.Code, body)
	}
	if !log.has("CreateUser:viewer") || !log.has("Login") {
		t.Fatalf("signup should create a viewer then sign in: %v", log.names)
	}
	if !strings.Contains(strings.Join(rec.Header().Values("Set-Cookie"), ";"), sessionCookieName+"=") {
		t.Fatalf("signup should set the session cookie: %v", rec.Header())
	}
	if identity, err := servicekit.VerifyToken(log.bearers[0], secret); err != nil || identity.Role != servicekit.RoleService {
		t.Fatalf("CreateUser should run as the BFF service: %+v %v", identity, err)
	}
	if log.bearers[1] != "" {
		t.Fatalf("Login must not carry the service token: %q", log.bearers[1])
	}

	// The browser cannot choose a role.
	if code, _ := do(t, h, "POST", "/api/auth/signup", "", `{"username":"boss","password":"long-enough-pass","role":"super_admin"}`); code != 400 {
		t.Fatalf("a role in the body must be rejected: %d", code)
	}
	if code, body := do(t, h, "POST", "/api/auth/signup", "", `{"username":"taken","password":"long-enough-pass"}`); code != 409 || field(body, "error") != "Username is already taken" {
		t.Fatalf("taken username: %d %v", code, body)
	}
	// Created, but the password did not log in (fake Login accepts only passwords starting "right").
	if code, body := do(t, h, "POST", "/api/auth/signup", "", `{"username":"fan2","password":"long-enough-pass"}`); code != 201 || field(body, "signed_in") != false {
		t.Fatalf("signup without login: %d %v", code, body)
	}
}

func TestSignupLimit(t *testing.T) {
	h, _, _ := newServer(t)
	last := 0
	for i := 0; i <= signupsPerMinute; i++ {
		last, _ = do(t, h, "POST", "/api/auth/signup", "", `{"username":"fan","password":"right-and-long"}`)
	}
	if last != 429 {
		t.Fatalf("sign-up %d in a minute should be refused, got %d", signupsPerMinute+1, last)
	}
	limiter := newWindowLimiter(1, time.Minute)
	now := time.Unix(0, 0)
	limiter.now = func() time.Time { return now }
	if !limiter.allow() || limiter.allow() {
		t.Fatal("limiter should allow exactly one event per window")
	}
	now = now.Add(time.Minute)
	if !limiter.allow() {
		t.Fatal("limiter should reset after the window")
	}
}

func TestCreateTeamRegistersIt(t *testing.T) {
	h, log, admin := newServer(t)
	code, body := do(t, h, "POST", "/api/teams", admin, `{"name":"Paro FC","dzongkhag":"Paro","tournament_id":"t1"}`)
	if code != 201 || field(body, "registration_status") != "approved" || !log.has("RegisterTeam:t1:new") || !log.has("ReviewRegistration:r1:approved") {
		t.Fatalf("admin entry should be registered and approved: %d %v %v", code, body, log.names)
	}

	h, log, _ = newServer(t)
	manager := roleToken(t, "m1", servicekit.RoleTeamManager)
	code, body = do(t, h, "POST", "/api/teams", manager, `{"name":"Paro FC","dzongkhag":"Paro","tournament_id":"t1"}`)
	if code != 201 || field(body, "registration_status") != "pending" || log.has("ReviewRegistration:r1:approved") {
		t.Fatalf("manager entry should wait for approval: %d %v %v", code, body, log.names)
	}

	code, body = do(t, h, "POST", "/api/teams", admin, `{"name":"Late FC","dzongkhag":"Paro","tournament_id":"closed"}`)
	if code != 201 || field(body, "registration_status") != "not_registered" || field(body, "registration_error") != "Registration is not open for this tournament" {
		t.Fatalf("team should be kept and the failure reported: %d %v", code, body)
	}
}

func TestRegistrationRoutes(t *testing.T) {
	h, log, admin := newServer(t)
	if code, _ := do(t, h, "GET", "/api/registrations", "", ""); code != 401 {
		t.Fatalf("registrations need sign-in: %d", code)
	}
	if code, body := do(t, h, "GET", "/api/registrations?tournament_id=t1&status=pending", admin, ""); code != 200 || field(body.([]any)[0], "status") != "pending" || !log.has("ListRegistrations:t1:pending") {
		t.Fatalf("list: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/registrations", admin, `{"tournament_id":"t1","team_id":"h"}`); code != 201 || field(body, "id") != "r1" {
		t.Fatalf("register: %d %v", code, body)
	}
	if code, _ := do(t, h, "PATCH", "/api/registrations/r1", admin, `{}`); code != 400 {
		t.Fatalf("approved is required: %d", code)
	}
	if code, body := do(t, h, "PATCH", "/api/registrations/r1", admin, `{"approved":true}`); code != 200 || field(body, "status") != "approved" {
		t.Fatalf("approve: %d %v", code, body)
	}
	if code, _ := do(t, h, "PATCH", "/api/registrations/r1", admin, `{"approved":false}`); code != 400 {
		t.Fatalf("reject without reason: %d", code)
	}
	if code, body := do(t, h, "PATCH", "/api/registrations/r1", admin, `{"approved":false,"reason":"Squad list missing"}`); code != 200 || field(body, "status") != "rejected" {
		t.Fatalf("reject: %d %v", code, body)
	}
}

func TestPlayerAndVenueRoutes(t *testing.T) {
	h, log, admin := newServer(t)
	if code, _ := do(t, h, "GET", "/api/teams/h/players", "", ""); code != 401 {
		t.Fatalf("players need sign-in: %d", code)
	}
	if code, body := do(t, h, "GET", "/api/teams/h/players", admin, ""); code != 200 || field(body.([]any)[0], "given_name") != "Sonam" {
		t.Fatalf("list players: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/teams/h/players", admin, `{"given_name":"Pema","family_name":"Wangchuk","shirt_number":"10"}`); code != 201 || field(body, "team_id") != "h" || !log.has("CreatePlayer:h") {
		t.Fatalf("create player: %d %v", code, body)
	}
	if code, body := do(t, h, "PATCH", "/api/players/p1", admin, `{"given_name":"Pema","family_name":"W","shirt_number":"7"}`); code != 200 || field(body, "shirt_number") != "7" {
		t.Fatalf("update player: %d %v", code, body)
	}
	if code, body := do(t, h, "GET", "/api/venues?dzongkhag=Thimphu", admin, ""); code != 200 || field(body.([]any)[0], "capacity") != float64(15000) || !log.has("ListVenues:Thimphu") {
		t.Fatalf("venues: %d %v", code, body)
	}
	if code, body := do(t, h, "POST", "/api/venues", admin, `{"name":" Paro Ground ","dzongkhag":"Paro","capacity":2000}`); code != 201 || field(body, "name") != "Paro Ground" {
		t.Fatalf("create venue: %d %v", code, body)
	}
}

func TestPreflightAllowsPatchForNewRoutes(t *testing.T) {
	h, _, _ := newServer(t)
	req := httptest.NewRequest("OPTIONS", "/api/registrations/r1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 || !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}
}
