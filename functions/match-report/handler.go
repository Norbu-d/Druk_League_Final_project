// Package main is the match-report OpenFaaS function. It receives a MatchFinalized event
// (forwarded by the function connector from NATS JetStream) and returns a short result
// announcement with the updated top of the table. It is stateless: names and standings are
// read from the BFF's public, read-only routes on every call.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// MatchFinalized is the event Score publishes on drukleague.match.finalized.
type MatchFinalized struct {
	EventID      string `json:"event_id"`
	MatchID      string `json:"match_id"`
	TournamentID string `json:"tournament_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeScore    int32  `json:"home_score"`
	AwayScore    int32  `json:"away_score"`
}

// Standing is one row of the public standings route.
type Standing struct {
	Team           string `json:"team"`
	Played         int    `json:"played"`
	Points         int    `json:"points"`
	GoalDifference int    `json:"goal_difference"`
	GoalsFor       int    `json:"goals_for"`
}

// Report is the function's response.
type Report struct {
	EventID    string     `json:"event_id"`
	MatchID    string     `json:"match_id"`
	Tournament string     `json:"tournament"`
	Headline   string     `json:"headline"`
	Text       string     `json:"text"`
	TopOfTable []Standing `json:"top_of_table"`
}

// Handler serves the function. API is the BFF base URL, for example http://bff:8080/api.
type Handler struct {
	API    string
	Client *http.Client
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "send the MatchFinalized event with POST")
		return
	}
	var event MatchFinalized
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	if err := decoder.Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "the body must be a MatchFinalized JSON event")
		return
	}
	if err := event.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	report := h.build(r.Context(), event)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func (e MatchFinalized) validate() error {
	switch {
	case !uuidPattern.MatchString(e.MatchID):
		return fmt.Errorf("match_id must be a UUID")
	case !uuidPattern.MatchString(e.TournamentID):
		return fmt.Errorf("tournament_id must be a UUID")
	case e.HomeScore < 0 || e.AwayScore < 0:
		return fmt.Errorf("scores cannot be negative")
	}
	return nil
}

// build looks up names and standings. Each lookup is best effort: if the API is down the
// report still carries the score, with generic names.
func (h Handler) build(ctx context.Context, e MatchFinalized) Report {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	home, away, venue := "Home team", "Away team", ""
	var matches []struct {
		ID    string `json:"id"`
		Home  string `json:"home"`
		Away  string `json:"away"`
		Venue string `json:"venue"`
	}
	if h.get(ctx, "/public/matches?tournament_id="+url.QueryEscape(e.TournamentID), &matches) == nil {
		for _, m := range matches {
			if m.ID == e.MatchID {
				home, away, venue = m.Home, m.Away, m.Venue
			}
		}
	}
	tournament := "the tournament"
	var tournaments []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if h.get(ctx, "/public/tournaments", &tournaments) == nil {
		for _, t := range tournaments {
			if t.ID == e.TournamentID && t.Name != "" {
				tournament = t.Name
			}
		}
	}
	table := h.standings(ctx, e.TournamentID, home, away)
	top := Rank(table)
	if len(top) > 3 {
		top = top[:3]
	}

	headline := fmt.Sprintf("%s %d–%d %s", home, e.HomeScore, e.AwayScore, away)
	text := "Full time"
	if venue != "" {
		text += " at " + venue
	}
	text += fmt.Sprintf(" in %s: %s. %s", tournament, headline, verdict(home, away, e.HomeScore, e.AwayScore))
	if len(top) > 0 {
		parts := make([]string, len(top))
		for i, row := range top {
			parts[i] = fmt.Sprintf("%d. %s (%d pts)", i+1, row.Team, row.Points)
		}
		text += " Top of the table: " + strings.Join(parts, ", ") + "."
	}
	return Report{EventID: e.EventID, MatchID: e.MatchID, Tournament: tournament, Headline: headline, Text: text, TopOfTable: top}
}

// standingsAttempts and standingsDelay bound the wait for the Standings projection. The
// function and Standings consume the same event independently, so the table can lag the
// announcement by a moment.
var (
	standingsAttempts = 5
	standingsDelay    = 400 * time.Millisecond
)

// standings returns the table once both teams of the match appear in it, retrying briefly;
// after the last attempt it returns whatever the API gave (possibly empty).
func (h Handler) standings(ctx context.Context, tournamentID, home, away string) []Standing {
	var table []Standing
	for attempt := 1; attempt <= standingsAttempts; attempt++ {
		table = nil
		if h.get(ctx, "/public/standings?tournament_id="+url.QueryEscape(tournamentID), &table) == nil && hasTeams(table, home, away) {
			return table
		}
		if attempt < standingsAttempts {
			select {
			case <-ctx.Done():
				return table
			case <-time.After(standingsDelay):
			}
		}
	}
	return table
}

func hasTeams(table []Standing, names ...string) bool {
	for _, name := range names {
		found := false
		for _, row := range table {
			if row.Team == name && row.Played > 0 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func verdict(home, away string, h, a int32) string {
	switch {
	case h > a:
		return home + " take the three points."
	case a > h:
		return away + " take the three points."
	}
	return "The points are shared."
}

// Rank orders standings by points, goal difference, goals scored, then name.
func Rank(rows []Standing) []Standing {
	ranked := append([]Standing(nil), rows...)
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

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
