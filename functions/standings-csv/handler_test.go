package main

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const tournamentID = "a1111111-1111-4111-8111-111111111111"

func fakeAPI(standingsStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/public/standings":
			w.WriteHeader(standingsStatus)
			_, _ = w.Write([]byte(`[{"team":"Haa","dzongkhag":"Haa","played":2,"won":1,"lost":1,"goals_for":3,"goals_against":3,"points":3},{"team":"=HYPERLINK(1)","dzongkhag":"Paro","played":2,"won":2,"goals_for":4,"goals_against":1,"goal_difference":3,"points":6}]`))
		case "/api/public/tournaments":
			_, _ = w.Write([]byte(`[{"id":"` + tournamentID + `","name":"Thimphu League 2026!"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func call(h http.Handler, method, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, "/"+query, nil))
	return rec
}

func TestDownloadsRankedCSV(t *testing.T) {
	api := fakeAPI(http.StatusOK)
	defer api.Close()
	rec := call(Handler{API: api.URL + "/api", Client: api.Client()}, http.MethodGet, "?tournament_id="+tournamentID)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="thimphu-league-2026-standings.csv"` {
		t.Fatalf("file name: %q", got)
	}
	records, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil || len(records) != 3 {
		t.Fatalf("csv: %v %v", records, err)
	}
	if records[1][0] != "1" || records[1][1] != "'=HYPERLINK(1)" || records[1][10] != "6" || records[2][1] != "Haa" {
		t.Fatalf("rows should be ranked and formula-safe: %v", records)
	}
}

func TestRejectsAndFailsClearly(t *testing.T) {
	api := fakeAPI(http.StatusServiceUnavailable)
	defer api.Close()
	h := Handler{API: api.URL + "/api", Client: api.Client()}
	if rec := call(h, http.MethodPost, "?tournament_id="+tournamentID); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: %d", rec.Code)
	}
	if rec := call(h, http.MethodGet, "?tournament_id=../../etc"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id: %d", rec.Code)
	}
	if rec := call(h, http.MethodGet, "?tournament_id="+tournamentID); rec.Code != http.StatusBadGateway {
		t.Fatalf("API down: %d", rec.Code)
	}
	api.Close()
	if rec := call(h, http.MethodGet, "?tournament_id="+tournamentID); rec.Code != http.StatusBadGateway {
		t.Fatalf("API unreachable: %d", rec.Code)
	}
}

func TestRankAndSlug(t *testing.T) {
	got := Rank([]Row{{Team: "B", Points: 1}, {Team: "A", Points: 1}, {Team: "C", Points: 1, GoalsFor: 1}, {Team: "D", Points: 1, GoalDifference: 2}})
	order := ""
	for _, r := range got {
		order += r.Team
	}
	if order != "DCAB" {
		t.Fatalf("rank order %q", order)
	}
	for in, want := range map[string]string{"": "drukleague", "!!!": "drukleague", "Paro  Cup--2026": "paro-cup-2026"} {
		if Slug(in) != want {
			t.Fatalf("Slug(%q) = %q, want %q", in, Slug(in), want)
		}
	}
	if safe("+1") != "'+1" || safe("Paro") != "Paro" || safe("") != "" {
		t.Fatal("safe() should guard formula cells only")
	}
}

func TestUnknownTournamentIsNotFound(t *testing.T) {
	api := fakeAPI(http.StatusOK)
	defer api.Close()
	rec := call(Handler{API: api.URL + "/api", Client: api.Client()}, http.MethodGet, "?tournament_id=b2222222-2222-4222-8222-222222222222")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown tournament: %d", rec.Code)
	}
}
