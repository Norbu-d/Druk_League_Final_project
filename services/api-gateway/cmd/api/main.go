package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
)

type principal struct {
	ID       string `json:"sub"`
	Username string `json:"username"`
	Role     string `json:"role"`

	Expires int64 `json:"exp"`
}
type server struct {
	db     *sql.DB
	secret []byte
	events *nats.Conn
}
type apiError struct {
	Error string `json:"error"`
}

func main() {
	addr := env("API_ADDR", ":8080")
	dsn := env("DATABASE_URL", "postgres://drukleague:drukleague@localhost:5432/drukleague?sslmode=disable")
	secret := env("JWT_SECRET", "local-development-secret-change-before-deployment")
	if env("APP_ENV", "development") == "production" && (secret == "local-development-secret-change-before-deployment" || env("BOOTSTRAP_ADMIN_PASSWORD", "drukleague-dev") == "drukleague-dev") {
		log.Fatal("JWT_SECRET and BOOTSTRAP_ADMIN_PASSWORD must be configured in production")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	for {
		if err = db.PingContext(ctx); err == nil {
			break
		}
		if ctx.Err() != nil {
			log.Fatalf("database unavailable: %v", err)
		}
		time.Sleep(time.Second)
	}
	if err = migrate(ctx, db, env("MIGRATION_FILE", "services/api-gateway/migrations/001_init.sql")); err != nil {
		log.Fatalf("migration: %v", err)
	}
	if err = bootstrap(ctx, db); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}
	appCtx, stop := context.WithCancel(context.Background())
	defer stop()
	s := &server{db: db, secret: []byte(secret)}
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		s.events, err = connectEvents(appCtx, db, natsURL)
		if err != nil {
			log.Fatalf("NATS connection: %v", err)
		}
		defer s.events.Drain()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("/api/", s.authenticate(http.HandlerFunc(s.api)))
	wrapped := cors(logging(mux))
	log.Printf("DrukLeague API listening on %s", addr)
	log.Fatal((&http.Server{Addr: addr, Handler: wrapped, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}).ListenAndServe())
}

func (s *server) api(w http.ResponseWriter, r *http.Request) {
	p := r.Context().Value("principal").(principal)
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		switch parts[0] {
		case "tournaments":
			if r.Method == http.MethodGet {
				s.listTournaments(w, r, p)
			} else if r.Method == http.MethodPost {
				s.createTournament(w, r, p)
			} else {
				methodNotAllowed(w)
			}
			return
		case "teams":
			if r.Method == http.MethodGet {
				s.listTeams(w, r, p)
			} else if r.Method == http.MethodPost {
				s.createTeam(w, r, p)
			} else {
				methodNotAllowed(w)
			}
			return
		case "matches":
			if r.Method == http.MethodPost {
				s.createMatch(w, r, p)
			} else if r.Method == http.MethodGet {
				s.listMatches(w, r, p)
			} else {
				methodNotAllowed(w)
			}
			return
		case "users":
			if r.Method == http.MethodGet {
				s.listUsers(w, r, p)
			} else if r.Method == http.MethodPost {
				s.createUser(w, r, p)
			} else {
				methodNotAllowed(w)
			}
			return
		case "standings":
			if r.Method == http.MethodGet {
				s.listStandings(w, r)
			} else {
				methodNotAllowed(w)
			}
			return
		}
	}
	if len(parts) == 3 && parts[0] == "matches" {
		switch parts[2] {
		case "official":
			if r.Method == http.MethodPatch {
				s.assignOfficial(w, r, p, parts[1])
			} else {
				methodNotAllowed(w)
			}
			return
		case "live":
			if r.Method == http.MethodPatch {
				s.updateLive(w, r, p, parts[1])
			} else {
				methodNotAllowed(w)
			}
			return
		case "finalize":
			if r.Method == http.MethodPost {
				s.finalize(w, r, p, parts[1])
			} else {
				methodNotAllowed(w)
			}
			return
		}
	}
	if len(parts) == 3 && parts[0] == "tournaments" && parts[2] == "fixtures" && r.Method == http.MethodPost {
		s.generateFixtures(w, r, p, parts[1])
		return
	}
	http.NotFound(w, r)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	var pwh, id, role string
	err := s.db.QueryRowContext(r.Context(), `SELECT id::text,password_hash,role FROM users WHERE username=$1`, strings.TrimSpace(in.Username)).Scan(&id, &pwh, &role)
	if err != nil || !verifyPassword(pwh, in.Password) {
		writeError(w, 401, "Invalid username or password")
		return
	}
	claim := principal{ID: id, Username: in.Username, Role: role, Expires: time.Now().Add(12 * time.Hour).Unix()}

	token, err := s.issue(claim)
	if err != nil {
		writeError(w, 500, "Could not create session")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "user": claim})
}
func (s *server) listTournaments(w http.ResponseWriter, r *http.Request, p principal) {
	rows, e := s.db.QueryContext(r.Context(), `SELECT id,name,sport,season,status,venue,starts_at,COALESCE(dzongkhag,'') FROM tournaments ORDER BY created_at DESC`)
	if e != nil {
		serverError(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, n, sp, se, st, v, dz string
		var d sql.NullTime
		if e = rows.Scan(&id, &n, &sp, &se, &st, &v, &d, &dz); e != nil {
			serverError(w, e)
			return
		}
		out = append(out, map[string]any{"id": id, "name": n, "sport": sp, "season": se, "status": st, "venue": v, "starts_at": nullableDate(d), "dzongkhag": dz})
	}
	writeJSON(w, 200, out)
}
func (s *server) createTournament(w http.ResponseWriter, r *http.Request, p principal) {
	if !canManage(p) {
		writeError(w, 403, "Only administrators can create tournaments")
		return
	}
	var in struct {
		Name      string `json:"name"`
		Sport     string `json:"sport"`
		Season    string `json:"season"`
		Venue     string `json:"venue"`
		StartsAt  string `json:"starts_at"`
		Dzongkhag string `json:"dzongkhag"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Dzongkhag = strings.TrimSpace(in.Dzongkhag)
	if in.Name == "" || in.Sport == "" || in.Season == "" || in.Dzongkhag == "" {
		writeError(w, 400, "name, sport, season, and dzongkhag are required")
		return
	}
	var districtExists bool
	if err := s.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM dzongkhags WHERE name=$1)`, in.Dzongkhag).Scan(&districtExists); err != nil {
		serverError(w, err)
		return
	}
	if !districtExists {
		writeError(w, 400, "dzongkhag must be one of the 20 Bhutanese districts")
		return
	}
	var id string
	e := s.db.QueryRowContext(r.Context(), `INSERT INTO tournaments(name,sport,season,venue,starts_at,created_by,dzongkhag) VALUES($1,$2,$3,$4,NULLIF($5,'')::date,NULLIF($6,'')::uuid,$7) RETURNING id::text`, in.Name, in.Sport, in.Season, in.Venue, in.StartsAt, p.ID, in.Dzongkhag).Scan(&id)
	if e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "name": in.Name, "sport": in.Sport, "season": in.Season, "venue": in.Venue, "dzongkhag": in.Dzongkhag})
}
func (s *server) listTeams(w http.ResponseWriter, r *http.Request, p principal) {
	q := `SELECT id::text,name,short_code,dzongkhag,COALESCE(tournament_id::text,'') FROM teams`
	args := []any{}

	q += ` ORDER BY name`
	rows, e := s.db.QueryContext(r.Context(), q, args...)
	if e != nil {
		serverError(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var id, n, sc, dz, tid string
		if e = rows.Scan(&id, &n, &sc, &dz, &tid); e != nil {
			serverError(w, e)
			return
		}
		out = append(out, map[string]string{"id": id, "name": n, "short_code": sc, "dzongkhag": dz, "tournament_id": tid})
	}
	writeJSON(w, 200, out)
}
func (s *server) createTeam(w http.ResponseWriter, r *http.Request, p principal) {
	if p.Role != "super_admin" {
		writeError(w, 403, "Only the tournament admin can register teams")
		return
	}
	var in struct {
		Name         string `json:"name"`
		Short        string `json:"short_code"`
		Dzongkhag    string `json:"dzongkhag"`
		TournamentID string `json:"tournament_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Dzongkhag = strings.TrimSpace(in.Dzongkhag)
	if in.Name == "" || in.Dzongkhag == "" {
		writeError(w, 400, "name and dzongkhag are required")
		return
	}

	if in.Short == "" {
		for _, v := range strings.Fields(in.Name) {
			if len(in.Short) < 3 {
				in.Short += strings.ToUpper(v[:1])
			}
		}
	}
	var districtExists bool
	if err := s.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM dzongkhags WHERE name=$1)`, in.Dzongkhag).Scan(&districtExists); err != nil {
		serverError(w, err)
		return
	}
	if !districtExists {
		writeError(w, 400, "dzongkhag must be one of the 20 Bhutanese districts")
		return
	}
	var id string
	e := s.db.QueryRowContext(r.Context(), `INSERT INTO teams(name,short_code,dzongkhag,tournament_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid) RETURNING id::text`, in.Name, in.Short, in.Dzongkhag, in.TournamentID).Scan(&id)
	if e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "name": in.Name, "short_code": in.Short, "dzongkhag": in.Dzongkhag})
}
func (s *server) listMatches(w http.ResponseWriter, r *http.Request, p principal) {
	q := `SELECT m.id::text,m.tournament_id::text,h.name,a.name,m.venue,m.kickoff,m.status,m.live_home,m.live_away,m.result_home,m.result_away FROM matches m JOIN teams h ON h.id=m.home_team_id JOIN teams a ON a.id=m.away_team_id`
	args := []any{}
	if id := r.URL.Query().Get("tournament_id"); id != "" {
		q += ` WHERE m.tournament_id=$1`
		args = append(args, id)
	}

	q += ` ORDER BY m.kickoff NULLS LAST`
	rows, e := s.db.QueryContext(r.Context(), q, args...)
	if e != nil {
		serverError(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, tid, home, away, venue, status string
		var kickoff sql.NullTime
		var lh, la, rh, ra sql.NullInt64
		if e = rows.Scan(&id, &tid, &home, &away, &venue, &kickoff, &status, &lh, &la, &rh, &ra); e != nil {
			serverError(w, e)
			return
		}
		out = append(out, map[string]any{"id": id, "tournament_id": tid, "home": home, "away": away, "venue": venue, "kickoff": nullableDate(kickoff), "status": status, "live_home": nullableInt(lh), "live_away": nullableInt(la), "result_home": nullableInt(rh), "result_away": nullableInt(ra)})
	}
	writeJSON(w, 200, out)
}
func (s *server) updateLive(w http.ResponseWriter, r *http.Request, p principal, id string) {
	if p.Role != "match_official" {
		writeError(w, 403, "Only the assigned match official can update scores")
		return
	}
	var in struct {
		Home int `json:"home"`
		Away int `json:"away"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Home < 0 || in.Away < 0 || in.Home > 99 || in.Away > 99 {
		writeError(w, 400, "Scores must be between 0 and 99")
		return
	}
	res, e := s.db.ExecContext(r.Context(), `UPDATE matches SET live_home=$1,live_away=$2,status='live' WHERE id=$3 AND finalized_at IS NULL AND official_id=$4::uuid`, in.Home, in.Away, id, p.ID)
	if e != nil {
		serverError(w, e)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 409, "Match is finalized, missing, or assigned to another official")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "live_home": in.Home, "live_away": in.Away, "status": "live"})
}
func (s *server) finalize(w http.ResponseWriter, r *http.Request, p principal, id string) {
	if p.Role != "match_official" {
		writeError(w, 403, "Only the assigned match official can finalize a result")
		return
	}
	tx, e := s.db.BeginTx(r.Context(), nil)
	if e != nil {
		serverError(w, e)
		return
	}
	defer tx.Rollback()
	var tid, homeID, awayID string
	var h, a int
	e = tx.QueryRowContext(r.Context(), `UPDATE matches SET result_home=COALESCE(live_home,0),result_away=COALESCE(live_away,0),finalized_at=now(),status='completed' WHERE id=$1 AND finalized_at IS NULL AND official_id=$2::uuid RETURNING tournament_id::text,home_team_id::text,away_team_id::text,result_home,result_away`, id, p.ID).Scan(&tid, &homeID, &awayID, &h, &a)
	if errors.Is(e, sql.ErrNoRows) {
		writeError(w, 409, "Match is already finalized, missing, or assigned to another official")
		return
	}
	if e != nil {
		serverError(w, e)
		return
	}
	for _, side := range []struct {
		id     string
		gf, ga int
	}{{homeID, h, a}, {awayID, a, h}} {
		points := 0
		if side.gf > side.ga {
			points = 3
		} else if side.gf == side.ga {
			points = 1
		}
		won, drawn, lost := 0, 0, 0
		if points == 3 {
			won = 1
		} else if points == 1 {
			drawn = 1
		} else {
			lost = 1
		}
		_, e = tx.ExecContext(r.Context(), `INSERT INTO standings(tournament_id,team_id,played,won,drawn,lost,goals_for,goals_against,points) VALUES($1,$2,1,$3,$4,$5,$6,$7,$8) ON CONFLICT(tournament_id,team_id) DO UPDATE SET played=standings.played+1,won=standings.won+EXCLUDED.won,drawn=standings.drawn+EXCLUDED.drawn,lost=standings.lost+EXCLUDED.lost,goals_for=standings.goals_for+EXCLUDED.goals_for,goals_against=standings.goals_against+EXCLUDED.goals_against,points=standings.points+EXCLUDED.points`, tid, side.id, won, drawn, lost, side.gf, side.ga, points)
		if e != nil {
			serverError(w, e)
			return
		}
	}
	var eventID string
	if e = tx.QueryRowContext(r.Context(), `SELECT gen_random_uuid()::text`).Scan(&eventID); e != nil {
		serverError(w, e)
		return
	}
	payload, e := json.Marshal(map[string]any{"event_id": eventID, "match_id": id, "tournament_id": tid, "home_team_id": homeID, "away_team_id": awayID, "home_score": h, "away_score": a, "finalized_at": time.Now().UTC()})
	if e != nil {
		serverError(w, e)
		return
	}
	if _, e = tx.ExecContext(r.Context(), `INSERT INTO event_outbox(subject,payload) VALUES('MatchFinalized',$1::jsonb)`, string(payload)); e != nil {
		serverError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": "completed", "home": h, "away": a})
}
func (s *server) listStandings(w http.ResponseWriter, r *http.Request) {
	tid := r.URL.Query().Get("tournament_id")
	if tid == "" {
		writeError(w, 400, "tournament_id is required")
		return
	}
	rows, e := s.db.QueryContext(r.Context(), `SELECT t.id::text,t.name,t.dzongkhag,s.played,s.won,s.drawn,s.lost,s.goals_for,s.goals_against,s.goals_for-s.goals_against,s.points FROM standings s JOIN teams t ON t.id=s.team_id WHERE s.tournament_id=$1 ORDER BY s.points DESC,(s.goals_for-s.goals_against) DESC,s.goals_for DESC,t.name`, tid)
	if e != nil {
		serverError(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, n, dz string
		var played, won, drawn, lost, gf, ga, gd, pts int
		if e = rows.Scan(&id, &n, &dz, &played, &won, &drawn, &lost, &gf, &ga, &gd, &pts); e != nil {
			serverError(w, e)
			return
		}
		out = append(out, map[string]any{"team_id": id, "team": n, "dzongkhag": dz, "played": played, "won": won, "drawn": drawn, "lost": lost, "goals_for": gf, "goals_against": ga, "goal_difference": gd, "points": pts})
	}
	writeJSON(w, 200, out)
}
func (s *server) generateFixtures(w http.ResponseWriter, r *http.Request, p principal, id string) {
	if !canManage(p) {
		writeError(w, 403, "Only administrators can generate fixtures")
		return
	}
	tx, e := s.db.BeginTx(r.Context(), nil)
	if e != nil {
		serverError(w, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), `SELECT pg_advisory_xact_lock(hashtext($1))`, id); e != nil {
		serverError(w, e)
		return
	}
	var count int
	if e = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM matches WHERE tournament_id=$1`, id).Scan(&count); e != nil {
		serverError(w, e)
		return
	}
	if count > 0 {
		writeError(w, 409, "Fixtures already exist for this tournament")
		return
	}
	teamQuery := `SELECT id::text FROM teams WHERE tournament_id=$1`
	teamArgs := []any{id}

	teamQuery += ` ORDER BY name`
	rows, e := tx.QueryContext(r.Context(), teamQuery, teamArgs...)
	if e != nil {
		serverError(w, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var team string
		if e = rows.Scan(&team); e != nil {
			rows.Close()
			serverError(w, e)
			return
		}
		ids = append(ids, team)
	}
	rows.Close()
	if len(ids) < 2 {
		writeError(w, 400, "At least two registered teams are required")
		return
	}
	if len(ids)%2 == 1 {
		ids = append(ids, "")
	}
	rounds := len(ids) - 1
	half := len(ids) / 2
	start := time.Now().Add(7 * 24 * time.Hour).Truncate(24 * time.Hour)
	made := 0
	for round := 0; round < rounds; round++ {
		for i := 0; i < half; i++ {
			home, away := ids[i], ids[len(ids)-1-i]
			if home != "" && away != "" {
				if (round+i)%2 == 1 {
					home, away = away, home
				}
				kick := start.Add(time.Duration(round*7+i) * 24 * time.Hour)
				_, e = tx.ExecContext(r.Context(), `INSERT INTO matches(tournament_id,home_team_id,away_team_id,kickoff) VALUES($1,$2,$3,$4)`, id, home, away, kick)
				if e != nil {
					serverError(w, e)
					return
				}
				made++
			}
		}
		last := ids[len(ids)-1]
		copy(ids[2:], ids[1:len(ids)-1])
		ids[1] = last
	}
	if e = tx.Commit(); e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 201, map[string]int{"fixtures_created": made})
}

func (s *server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		p, e := s.verify(a)
		if e != nil {
			writeError(w, 401, "Sign in to continue")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "principal", p)))
	})
}
func (s *server) issue(p principal) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, e := json.Marshal(p)
	if e != nil {
		return "", e
	}
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (s *server) verify(t string) (principal, error) {
	var p principal
	parts := strings.Split(t, ".")
	if len(parts) != 3 {
		return p, errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return p, errors.New("bad signature")
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || json.Unmarshal(b, &p) != nil || p.Expires < time.Now().Unix() {
		return p, errors.New("expired token")
	}
	return p, nil
}
func bootstrap(ctx context.Context, db *sql.DB) error {
	user := env("BOOTSTRAP_ADMIN_USER", "admin")
	pass := env("BOOTSTRAP_ADMIN_PASSWORD", "drukleague-dev")
	hash := hashPassword(pass)
	_, e := db.ExecContext(ctx, `INSERT INTO users(username,password_hash,role) VALUES($1,$2,'super_admin') ON CONFLICT(username) DO NOTHING`, user, hash)
	return e
}
func hashPassword(p string) string {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		panic(e)
	}
	derived := pbkdf2([]byte(p), salt, 120000, 32)
	return fmt.Sprintf("pbkdf2$120000$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived))
}
func verifyPassword(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iterations := 120000
	var salt, want []byte
	salt, _ = base64.RawStdEncoding.DecodeString(parts[2])
	want, _ = base64.RawStdEncoding.DecodeString(parts[3])
	got := pbkdf2([]byte(password), salt, iterations, len(want))
	return hmac.Equal(got, want)
}
func pbkdf2(password, salt []byte, iter, keyLen int) []byte {
	out := make([]byte, 0, keyLen)
	for block := uint32(1); len(out) < keyLen; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
func migrate(ctx context.Context, db *sql.DB, path string) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	_, e = db.ExecContext(ctx, string(b))
	return e
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		writeError(w, 400, "Invalid request body")
		return false
	}
	return true
}
func canManage(p principal) bool { return p.Role == "super_admin" }
func nullableDate(v sql.NullTime) any {
	if !v.Valid {
		return nil
	}
	return v.Time
}
func nullableInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
}
func serverError(w http.ResponseWriter, e error) {
	log.Printf("request failed: %v", e)
	writeError(w, 500, "The request could not be completed")
}
func methodNotAllowed(w http.ResponseWriter) { writeError(w, 405, "Method not allowed") }
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", env("CORS_ORIGIN", "*"))
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
