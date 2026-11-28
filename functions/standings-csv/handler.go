// Package main is the standings-csv OpenFaaS function: GET ?tournament_id=<uuid> returns the
// tournament's league table as a CSV download. It is stateless and reads only the BFF's
// public, read-only routes.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Row is one team in the public standings response.
type Row struct {
	Team           string `json:"team"`
	Dzongkhag      string `json:"dzongkhag"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
}

// Handler serves the function. API is the BFF base URL, for example http://bff:8080/api.
type Handler struct {
	API    string
	Client *http.Client
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET ?tournament_id=<id>", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("tournament_id")
	if !uuidPattern.MatchString(id) {
		http.Error(w, "tournament_id must be a UUID", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var rows []Row
	if err := h.get(ctx, "/public/standings?tournament_id="+url.QueryEscape(id), &rows); err != nil {
		http.Error(w, "standings are unavailable right now", http.StatusBadGateway)
		return
	}
	name := "tournament"
	var tournaments []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if h.get(ctx, "/public/tournaments", &tournaments) == nil {
		for _, t := range tournaments {
			if t.ID == id && t.Name != "" {
				name = t.Name
			}
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-standings.csv"`, Slug(name)))
	w.Header().Set("Cache-Control", "no-store")
	_ = WriteCSV(w, Rank(rows))
}

// Rank orders rows by points, goal difference, goals scored, then team name.
func Rank(rows []Row) []Row {
	ranked := append([]Row(nil), rows...)
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GoalDifference != b.GoalDifference {
			return a.GoalDifference > b.GoalDifference
		}
		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}
		return a.Team < b.Team
	})
	return ranked
}

// WriteCSV writes the ranked table. Text cells that a spreadsheet would treat as a formula
// are prefixed with an apostrophe.
func WriteCSV(w io.Writer, rows []Row) error {
	out := csv.NewWriter(w)
	_ = out.Write([]string{"Rank", "Team", "Dzongkhag", "Played", "Won", "Drawn", "Lost", "Goals for", "Goals against", "Goal difference", "Points"})
	for i, r := range rows {
		n := func(v int) string { return strconv.Itoa(v) }
		_ = out.Write([]string{n(i + 1), safe(r.Team), safe(r.Dzongkhag), n(r.Played), n(r.Won), n(r.Drawn), n(r.Lost), n(r.GoalsFor), n(r.GoalsAgainst), n(r.GoalDifference), n(r.Points)})
	}
	out.Flush()
	return out.Error()
}

func safe(text string) string {
	if text != "" && strings.ContainsRune("=+-@\t\r", rune(text[0])) {
		return "'" + text
	}
	return text
}

// Slug turns a tournament name into a safe file name.
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "drukleague"
	}
	return slug
}

func (h Handler) get(ctx context.Context, path string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.API, "/")+path, nil)
	if err != nil {
		return err
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", path, resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(into)
}
