package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	tournamentID = "a1111111-1111-4111-8111-111111111111"
	matchID      = "c1111111-1111-4111-8111-111111111111"
)

// fakeAPI serves the three public BFF routes the function reads.
func fakeAPI(t *testing.T, standings string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/public/matches":
			if r.URL.Query().Get("tournament_id") != tournamentID {
				t.Errorf("matches should be filtered by tournament, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"id":"` + matchID + `","home":"Paro United","away":"Thimphu City","venue":"Changlimithang"}]`))
		case "/api/public/tournaments":
			_, _ = w.Write([]byte(`[{"id":"` + tournamentID + `","name":"Thimphu League"}]`))
		case "/api/public/standings":
			_, _ = w.Write([]byte(standings))
		default:
			http.NotFound(w, r)
		}
	}))
}

func invoke(t *testing.T, h http.Handler, method, body string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, "/", strings.NewReader(body)))
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func event(home, away int) string {
	b, _ := json.Marshal(MatchFinalized{EventID: "e1", MatchID: matchID, TournamentID: tournamentID, HomeScore: int32(home), AwayScore: int32(away)})
	return string(b)
}

func TestReportUsesNamesVenueAndTopOfTable(t *testing.T) {
	api := fakeAPI(t, `[{"team":"Thimphu City","points":3,"goal_difference":1,"goals_for":2},{"team":"Paro United","points":6,"goal_difference":3,"goals_for":5},{"team":"Haa","points":3,"goal_difference":1,"goals_for":3},{"team":"Bumthang","points":0}]`)
	defer api.Close()
	code, out := invoke(t, Handler{API: api.URL + "/api/", Client: api.Client()}, http.MethodPost, event(2, 1))
	if code != 200 || out["headline"] != "Paro United 2–1 Thimphu City" || out["tournament"] != "Thimphu League" {
		t.Fatalf("report: %d %v", code, out)
	}
	text := out["text"].(string)
	for _, want := range []string{"at Changlimithang", "Paro United take the three points", "1. Paro United (6 pts), 2. Haa (3 pts), 3. Thimphu City (3 pts)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text %q should contain %q", text, want)
		}
	}
	if len(out["top_of_table"].([]any)) != 3 {
		t.Fatalf("top of table should hold three rows: %v", out["top_of_table"])
	}
}

func TestVerdicts(t *testing.T) {
	api := fakeAPI(t, `[]`)
	defer api.Close()
	h := Handler{API: api.URL + "/api", Client: api.Client()}
	if _, out := invoke(t, h, http.MethodPost, event(0, 3)); !strings.Contains(out["text"].(string), "Thimphu City take the three points") {
		t.Fatalf("away win: %v", out["text"])
	}
	if _, out := invoke(t, h, http.MethodPost, event(1, 1)); !strings.Contains(out["text"].(string), "The points are shared") || strings.Contains(out["text"].(string), "Top of the table") {
		t.Fatalf("draw without standings: %v", out["text"])
	}
}

func TestReportDegradesWhenTheAPIIsDown(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer api.Close()
	code, out := invoke(t, Handler{API: api.URL, Client: api.Client()}, http.MethodPost, event(3, 0))
	if code != 200 || out["headline"] != "Home team 3–0 Away team" || out["tournament"] != "the tournament" {
		t.Fatalf("degraded report: %d %v", code, out)
	}
	// Unreachable API (closed server) is also tolerated.
	api.Close()
	if code, _ := invoke(t, Handler{API: api.URL, Client: api.Client()}, http.MethodPost, event(1, 0)); code != 200 {
		t.Fatalf("closed API: %d", code)
	}
}

func TestRejectsBadInvocations(t *testing.T) {
	h := Handler{API: "http://127.0.0.1:1", Client: http.DefaultClient}
	cases := []struct {
		method, body string
		code         int
	}{
		{http.MethodGet, "", http.StatusMethodNotAllowed},
		{http.MethodPost, "not json", http.StatusBadRequest},
		{http.MethodPost, `{"match_id":"x","tournament_id":"` + tournamentID + `"}`, http.StatusBadRequest},
		{http.MethodPost, `{"match_id":"` + matchID + `","tournament_id":"y"}`, http.StatusBadRequest},
		{http.MethodPost, `{"match_id":"` + matchID + `","tournament_id":"` + tournamentID + `","home_score":-1}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		if code, out := invoke(t, h, c.method, c.body); code != c.code || out["error"] == nil {
			t.Fatalf("%s %q: got %d %v", c.method, c.body, code, out)
		}
	}
}

func TestRankTieBreaks(t *testing.T) {
	got := Rank([]Standing{{Team: "B", Points: 3}, {Team: "A", Points: 3}, {Team: "C", Points: 3, GoalsFor: 2}, {Team: "D", Points: 3, GoalDifference: 1}})
	order := ""
	for _, row := range got {
		order += row.Team
	}
	if order != "DCAB" {
		t.Fatalf("rank order %q, want DCAB (points, goal difference, goals for, name)", order)
	}
}

func init() { standingsDelay = time.Millisecond }

func TestWaitsForTheStandingsProjection(t *testing.T) {
	calls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/public/matches":
			_, _ = w.Write([]byte(`[{"id":"` + matchID + `","home":"Paro United","away":"Thimphu City"}]`))
		case "/api/public/standings":
			calls++
			if calls < 3 { // the projection has not applied the event yet
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"team":"Paro United","played":1,"points":3},{"team":"Thimphu City","played":1,"points":0}]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer api.Close()
	_, out := invoke(t, Handler{API: api.URL + "/api", Client: api.Client()}, http.MethodPost, event(2, 1))
	if calls != 3 || !strings.Contains(out["text"].(string), "1. Paro United (3 pts)") {
		t.Fatalf("should retry until both teams are in the table: %d calls, %v", calls, out["text"])
	}
}

func TestStopsWaitingAfterTheLastAttempt(t *testing.T) {
	calls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/standings" {
			calls++
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer api.Close()
	if code, _ := invoke(t, Handler{API: api.URL + "/api", Client: api.Client()}, http.MethodPost, event(0, 0)); code != 200 || calls != standingsAttempts {
		t.Fatalf("expected %d attempts then a report, got %d calls (status %d)", standingsAttempts, calls, code)
	}
}
