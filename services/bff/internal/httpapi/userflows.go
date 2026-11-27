package httpapi

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	playerv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"google.golang.org/grpc/status"
)

// --- Sign-up

// windowLimiter allows at most max events per window across all callers.
type windowLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	start  time.Time
	count  int
	now    func() time.Time
}

func newWindowLimiter(max int, window time.Duration) *windowLimiter {
	return &windowLimiter{max: max, window: window, now: time.Now}
}

func (l *windowLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.start) >= l.window {
		l.start, l.count = now, 0
	}
	if l.count >= l.max {
		return false
	}
	l.count++
	return true
}

// signup creates a viewer account and signs it in. The role is fixed here: the browser
// cannot choose it, and the user service only lets the BFF's service identity create
// viewers and team managers. Team managers and officials are still added by the admin.
func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	if !s.signups.allow() {
		writeError(w, http.StatusTooManyRequests, "Too many new accounts right now; try again in a minute")
		return
	}
	token, err := s.tokens.Token()
	if err != nil {
		writeError(w, 500, "The request could not be completed")
		return
	}
	username := strings.TrimSpace(in.Username)
	created, err := s.c.Users.CreateUser(servicekit.WithBearer(r.Context(), token), &v1.CreateUserRequest{Username: username, Password: in.Password, Role: servicekit.RoleViewer})
	if err != nil {
		writeStatus(w, err)
		return
	}
	user := created.GetUser()
	profile := map[string]string{"sub": user.GetId(), "username": user.GetUsername(), "role": user.GetRole()}
	// Login is a public RPC; it runs on the plain request context, without the service token.
	login, err := s.c.Auth.Login(r.Context(), &authv1.LoginRequest{Username: username, Password: in.Password})
	if err != nil {
		writeJSON(w, 201, map[string]any{"user": profile, "signed_in": false})
		return
	}
	var expiresAt time.Time
	if login.GetExpiresAt() != nil {
		expiresAt = login.GetExpiresAt().AsTime()
	}
	setAuthCookies(w, login.GetAccessToken(), login.GetRefreshToken(), expiresAt)
	writeJSON(w, 201, map[string]any{"user": profile, "signed_in": true})
}

// --- Registrations

func registrationJSON(reg *registrationv1.Registration) map[string]any {
	return map[string]any{"id": reg.GetId(), "tournament_id": reg.GetTournamentId(), "team_id": reg.GetTeamId(), "status": reg.GetStatus(), "submitted_at": timeOrNil(reg.GetSubmittedAt())}
}

// enterTeam registers a team for a tournament and, when the admin entered it, approves it.
// It returns the registration status ("approved", "pending" or "not_registered") and, when
// a step failed, the reason to show.
func (s *Server) enterTeam(ctx context.Context, tournamentID, teamID string) (string, string) {
	out, err := s.c.Registrations.RegisterTeam(ctx, &registrationv1.RegisterTeamRequest{TournamentId: tournamentID, TeamId: teamID})
	if err != nil {
		st, _ := status.FromError(err)
		return "not_registered", friendly(st.Message())
	}
	reg := out.GetRegistration()
	if identity, ok := servicekit.IdentityFromContext(ctx); ok && identity.Role == servicekit.RoleSuperAdmin {
		reviewed, err := s.c.Registrations.ReviewRegistration(ctx, &registrationv1.ReviewRegistrationRequest{RegistrationId: reg.GetId(), Approved: true})
		if err != nil {
			st, _ := status.FromError(err)
			return reg.GetStatus(), friendly(st.Message())
		}
		return reviewed.GetRegistration().GetStatus(), ""
	}
	return reg.GetStatus(), ""
}

// listRegistrations: the admin sees every registration, a team manager only their own
// (the registration service applies that filter).
func (s *Server) listRegistrations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := collect(r.Context(), func(ctx context.Context, token string) ([]*registrationv1.Registration, string, error) {
		out, err := s.c.Registrations.ListRegistrations(ctx, &registrationv1.ListRegistrationsRequest{TournamentId: q.Get("tournament_id"), Status: q.Get("status"), PageSize: 100, PageToken: token})
		return out.GetRegistrations(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, reg := range items {
		result = append(result, registrationJSON(reg))
	}
	writeJSON(w, 200, result)
}

func (s *Server) registerTeam(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TournamentID string `json:"tournament_id"`
		TeamID       string `json:"team_id"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Registrations.RegisterTeam(r.Context(), &registrationv1.RegisterTeamRequest{TournamentId: strings.TrimSpace(in.TournamentID), TeamId: strings.TrimSpace(in.TeamID)})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, registrationJSON(out.GetRegistration()))
}

func (s *Server) reviewRegistration(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Approved *bool  `json:"approved"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	if in.Approved == nil {
		writeError(w, 400, "approved is required (true or false)")
		return
	}
	out, err := s.c.Registrations.ReviewRegistration(r.Context(), &registrationv1.ReviewRegistrationRequest{RegistrationId: r.PathValue("id"), Approved: *in.Approved, Reason: strings.TrimSpace(in.Reason)})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 200, registrationJSON(out.GetRegistration()))
}

// --- Players

func playerJSON(p *playerv1.Player) map[string]string {
	return map[string]string{"id": p.GetId(), "team_id": p.GetTeamId(), "given_name": p.GetGivenName(), "family_name": p.GetFamilyName(), "shirt_number": p.GetShirtNumber()}
}

type playerBody struct {
	GivenName   string `json:"given_name"`
	FamilyName  string `json:"family_name"`
	ShirtNumber string `json:"shirt_number"`
}

func (s *Server) listPlayers(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	items, err := collect(r.Context(), func(ctx context.Context, token string) ([]*playerv1.Player, string, error) {
		out, err := s.c.Players.ListPlayers(ctx, &playerv1.ListPlayersRequest{TeamId: teamID, PageSize: 100, PageToken: token})
		return out.GetPlayers(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]string, 0, len(items))
	for _, p := range items {
		result = append(result, playerJSON(p))
	}
	writeJSON(w, 200, result)
}

// createPlayer: the player service allows the admin, or the manager of this team.
func (s *Server) createPlayer(w http.ResponseWriter, r *http.Request) {
	var in playerBody
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Players.CreatePlayer(r.Context(), &playerv1.CreatePlayerRequest{TeamId: r.PathValue("id"), GivenName: in.GivenName, FamilyName: in.FamilyName, ShirtNumber: in.ShirtNumber})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, playerJSON(out.GetPlayer()))
}

func (s *Server) updatePlayer(w http.ResponseWriter, r *http.Request) {
	var in playerBody
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Players.UpdatePlayer(r.Context(), &playerv1.UpdatePlayerRequest{Id: r.PathValue("id"), GivenName: in.GivenName, FamilyName: in.FamilyName, ShirtNumber: in.ShirtNumber})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 200, playerJSON(out.GetPlayer()))
}

// --- Venues

func venueJSON(v *venuev1.Venue) map[string]any {
	return map[string]any{"id": v.GetId(), "name": v.GetName(), "dzongkhag": v.GetDzongkhag(), "capacity": v.GetCapacity()}
}

func (s *Server) listVenues(w http.ResponseWriter, r *http.Request) {
	dzongkhag := r.URL.Query().Get("dzongkhag")
	items, err := collect(r.Context(), func(ctx context.Context, token string) ([]*venuev1.Venue, string, error) {
		out, err := s.c.Venues.ListVenues(ctx, &venuev1.ListVenuesRequest{Dzongkhag: dzongkhag, PageSize: 100, PageToken: token})
		return out.GetVenues(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, v := range items {
		result = append(result, venueJSON(v))
	}
	writeJSON(w, 200, result)
}

// createVenue: the venue service allows only the admin. Fixture generation books the
// venues in the tournament's dzongkhag.
func (s *Server) createVenue(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string `json:"name"`
		Dzongkhag string `json:"dzongkhag"`
		Capacity  int32  `json:"capacity"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Venues.CreateVenue(r.Context(), &venuev1.CreateVenueRequest{Name: strings.TrimSpace(in.Name), Dzongkhag: strings.TrimSpace(in.Dzongkhag), Capacity: in.Capacity})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, venueJSON(out.GetVenue()))
}
