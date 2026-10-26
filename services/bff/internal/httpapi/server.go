// Package httpapi is the backend-for-frontend: the same REST routes and JSON shapes as the
// monolith (services/api-gateway), served by calling the gRPC microservices instead of one
// shared database. The browser keeps working unchanged; the gateway validates the JWT once
// and forwards the caller's token so every service still applies its own role checks.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	officialv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Clients are the services the gateway composes.
type Clients struct {
	Auth        authv1.AuthServiceClient
	Users       v1.UserServiceClient
	Tournaments v1.TournamentServiceClient
	Teams       v1.TeamServiceClient
	Matches     v1.MatchServiceClient
	Scores      v1.ScoreServiceClient
	Standings   standingsv1.StandingsServiceClient
	Fixtures    fixturev1.FixtureServiceClient
	Officials   officialv1.OfficialServiceClient
}

type Server struct {
	c          Clients
	secret     []byte
	corsOrigin string
}

func New(clients Clients, secret []byte, corsOrigin string) *Server {
	if corsOrigin == "" {
		corsOrigin = "*"
	}
	return &Server{c: clients, secret: secret, corsOrigin: corsOrigin}
}

// Handler returns the routes, wrapped in CORS, request-id and access logging.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/refresh", s.refresh)
	routes := map[string]http.HandlerFunc{
		"POST /api/auth/logout":               s.logout,
		"GET /api/tournaments":                s.listTournaments,
		"POST /api/tournaments":               s.createTournament,
		"POST /api/tournaments/{id}/fixtures": s.generateFixtures,
		"GET /api/teams":                      s.listTeams,
		"POST /api/teams":                     s.createTeam,
		"GET /api/matches":                    s.listMatches,
		"POST /api/matches":                   s.createMatch,
		"PATCH /api/matches/{id}/official":    s.assignOfficial,
		"PATCH /api/matches/{id}/live":        s.updateLive,
		"POST /api/matches/{id}/finalize":     s.finalize,
		"GET /api/standings":                  s.listStandings,
		"GET /api/users":                      s.listUsers,
		"POST /api/users":                     s.createUser,
	}
	for pattern, handler := range routes {
		mux.Handle(pattern, s.authenticate(handler))
	}
	return s.cors(logging(mux))
}

// authenticate verifies the bearer token once at the edge and marks the context so outbound
// gRPC calls forward it.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeError(w, 401, "Sign in required")
			return
		}
		if _, err := servicekit.VerifyToken(token, s.secret); err != nil {
			writeError(w, 401, "Session expired or invalid; sign in again")
			return
		}
		next.ServeHTTP(w, r.WithContext(servicekit.WithBearer(r.Context(), token)))
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logging gives every request an id that is propagated to the services, so one browser
// action can be followed through all the service logs.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" {
			id = servicekit.NewRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		recorder := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(recorder, r.WithContext(servicekit.WithRequestID(r.Context(), id)))
		log.Printf("request_id=%s %s %s %d %s", id, r.Method, r.URL.Path, recorder.status, time.Since(start).Round(time.Millisecond))
	})
}

// decode reads a JSON body strictly, like the monolith. An empty body is allowed only when
// optional is true.
func decode(w http.ResponseWriter, r *http.Request, v any, optional bool) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil && !(optional && errors.Is(err, io.EOF)) {
		writeError(w, 400, "Invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

// writeStatus turns a service's gRPC error into the HTTP status and message the browser shows.
func writeStatus(w http.ResponseWriter, err error) {
	st, _ := status.FromError(err)
	code := map[codes.Code]int{
		codes.InvalidArgument:    400,
		codes.Unauthenticated:    401,
		codes.PermissionDenied:   403,
		codes.NotFound:           404,
		codes.AlreadyExists:      409,
		codes.FailedPrecondition: 409,
		codes.Unavailable:        503,
		codes.DeadlineExceeded:   504,
	}[st.Code()]
	if code == 0 {
		log.Printf("service error: %v", err)
		writeError(w, 500, "The request could not be completed")
		return
	}
	writeError(w, code, friendly(st.Message()))
}

// friendly drops the "invalid request: " style prefix the services put before their message.
func friendly(message string) string {
	if _, rest, ok := strings.Cut(message, ": "); ok && rest != "" {
		message = rest
	}
	if message == "" {
		return "Request failed"
	}
	return strings.ToUpper(message[:1]) + message[1:]
}

// maxPages bounds list loops so a misbehaving page token cannot spin forever.
const maxPages = 50

// collect walks every page of a List RPC.
func collect[T any](ctx context.Context, page func(ctx context.Context, token string) ([]T, string, error)) ([]T, error) {
	var all []T
	token := ""
	for i := 0; i < maxPages; i++ {
		items, next, err := page(ctx, token)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if next == "" {
			return all, nil
		}
		token = next
	}
	return all, nil
}
