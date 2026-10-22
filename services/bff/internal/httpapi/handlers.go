package httpapi

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	officialv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Auth

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Auth.Login(r.Context(), &authv1.LoginRequest{Username: strings.TrimSpace(in.Username), Password: in.Password})
	if err != nil {
		writeStatus(w, err)
		return
	}
	user := out.GetUser()
	writeJSON(w, 200, map[string]any{
		"token":         out.GetAccessToken(),
		"refresh_token": out.GetRefreshToken(),
		"expires_at":    timeOrNil(out.GetExpiresAt()),
		"user":          map[string]any{"sub": user.GetId(), "username": user.GetUsername(), "role": user.GetRole(), "exp": out.GetExpiresAt().GetSeconds()},
	})
}

// refresh swaps a refresh token for a new access token (and a rotated refresh token), so
// the browser does not lose its session when the 30-minute access token expires.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Auth.RefreshToken(r.Context(), &authv1.RefreshTokenRequest{RefreshToken: in.RefreshToken})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"token": out.GetAccessToken(), "refresh_token": out.GetRefreshToken(), "expires_at": timeOrNil(out.GetExpiresAt())})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if _, err := s.c.Auth.RevokeSession(r.Context(), &authv1.RevokeSessionRequest{}); err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"revoked": true})
}

// --- Users

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	users, err := collect(r.Context(), func(ctx context.Context, token string) ([]*v1.User, string, error) {
		out, err := s.c.Users.ListUsers(ctx, &v1.ListUsersRequest{Role: role, PageSize: 100, PageToken: token})
		return out.GetUsers(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]string, 0, len(users))
	for _, u := range users {
		result = append(result, map[string]string{"id": u.GetId(), "username": u.GetUsername(), "role": u.GetRole()})
	}
	writeJSON(w, 200, result)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Users.CreateUser(r.Context(), &v1.CreateUserRequest{Username: in.Username, Password: in.Password, Role: in.Role})
	if err != nil {
		writeStatus(w, err)
		return
	}
	u := out.GetUser()
	writeJSON(w, 201, map[string]string{"id": u.GetId(), "username": u.GetUsername(), "role": u.GetRole()})
}

// --- Tournaments

func tournamentJSON(t *v1.Tournament) map[string]any {
	return map[string]any{"id": t.GetId(), "name": t.GetName(), "sport": t.GetSport(), "season": t.GetSeason(), "status": t.GetStatus(), "venue": t.GetVenue(), "starts_at": timeOrNil(t.GetStartsAt()), "dzongkhag": t.GetDzongkhag()}
}

func (s *Server) listTournaments(w http.ResponseWriter, r *http.Request) {
	items, err := collect(r.Context(), func(ctx context.Context, token string) ([]*v1.Tournament, string, error) {
		out, err := s.c.Tournaments.ListTournaments(ctx, &v1.ListTournamentsRequest{Season: r.URL.Query().Get("season"), PageSize: 100, PageToken: token})
		return out.GetTournaments(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, t := range items {
		result = append(result, tournamentJSON(t))
	}
	writeJSON(w, 200, result)
}

func (s *Server) createTournament(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string `json:"name"`
		Sport     string `json:"sport"`
		Season    string `json:"season"`
		Venue     string `json:"venue"`
		StartsAt  string `json:"starts_at"`
		Dzongkhag string `json:"dzongkhag"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	req := &v1.CreateTournamentRequest{Name: in.Name, Sport: in.Sport, Season: in.Season, Venue: in.Venue, Dzongkhag: in.Dzongkhag}
	if starts := strings.TrimSpace(in.StartsAt); starts != "" {
		t, ok := parseDateOrTime(starts)
		if !ok {
			writeError(w, 400, "starts_at must be a date (YYYY-MM-DD) or an RFC3339 timestamp")
			return
		}
		req.StartsAt = timestamppb.New(t)
	}
	out, err := s.c.Tournaments.CreateTournament(r.Context(), req)
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, tournamentJSON(out.GetTournament()))
}

// generateFixtures starts the schedule on the tournament's own start date (10:00 Bhutan
// time) when it is still in the future; otherwise Fixture's default of one week ahead applies.
func (s *Server) generateFixtures(w http.ResponseWriter, r *http.Request) {
	req := &fixturev1.GenerateFixturesRequest{TournamentId: r.PathValue("id")}
	tournament, err := s.c.Tournaments.GetTournament(r.Context(), &v1.GetTournamentRequest{Id: req.TournamentId})
	if err != nil {
		writeStatus(w, err)
		return
	}
	if starts := tournament.GetTournament().GetStartsAt(); starts != nil {
		day := starts.AsTime().UTC()
		if first := time.Date(day.Year(), day.Month(), day.Day(), 4, 0, 0, 0, time.UTC); first.After(time.Now()) {
			req.StartsAt = timestamppb.New(first)
		}
	}
	out, err := s.c.Fixtures.GenerateFixtures(r.Context(), req)
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, map[string]int32{"fixtures_created": out.GetCreatedCount()})
}

// --- Teams

func teamJSON(t *v1.Team) map[string]string {
	return map[string]string{"id": t.GetId(), "name": t.GetName(), "short_code": t.GetShortCode(), "dzongkhag": t.GetDzongkhag(), "tournament_id": t.GetTournamentId(), "manager_id": t.GetManagerId()}
}

func (s *Server) listTeams(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := collect(r.Context(), func(ctx context.Context, token string) ([]*v1.Team, string, error) {
		out, err := s.c.Teams.ListTeams(ctx, &v1.ListTeamsRequest{TournamentId: q.Get("tournament_id"), Dzongkhag: q.Get("dzongkhag"), ManagerId: q.Get("manager_id"), PageSize: 100, PageToken: token})
		return out.GetTeams(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]string, 0, len(items))
	for _, t := range items {
		result = append(result, teamJSON(t))
	}
	writeJSON(w, 200, result)
}

func (s *Server) createTeam(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name         string `json:"name"`
		Short        string `json:"short_code"`
		Dzongkhag    string `json:"dzongkhag"`
		TournamentID string `json:"tournament_id"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	out, err := s.c.Teams.CreateTeam(r.Context(), &v1.CreateTeamRequest{Name: in.Name, ShortCode: shortCode(in.Short, in.Name), Dzongkhag: in.Dzongkhag, TournamentId: in.TournamentID})
	if err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 201, teamJSON(out.GetTeam()))
}

// shortCode keeps the caller's code when it is usable. The web form sends the name's
// initials, which is a single letter for a one-word name such as "Paro", so fall back to
// the initials of a longer name or the first three letters of a one-word name.
func shortCode(given, name string) string {
	clean := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToUpper(r)
			}
			return -1
		}, s)
	}
	if code := clean(given); len(code) >= 2 {
		return code
	}
	words := strings.Fields(name)
	initials := ""
	for _, word := range words {
		if c := clean(word); c != "" && len(initials) < 4 {
			initials += c[:1]
		}
	}
	if len(initials) >= 2 {
		return initials
	}
	code := clean(name)
	if len(code) > 3 {
		code = code[:3]
	}
	return code
}

// --- Matches and scores

// listMatches joins Match (fixture), Team (names) and Score (live or final result) into the
// row shape the monolith returned. A Team or Score outage degrades the row instead of
// failing the whole list.
func (s *Server) listMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tournamentID := r.URL.Query().Get("tournament_id")
	items, err := collect(ctx, func(ctx context.Context, token string) ([]*v1.Match, string, error) {
		out, err := s.c.Matches.ListMatches(ctx, &v1.ListMatchesRequest{TournamentId: tournamentID, PageSize: 100, PageToken: token})
		return out.GetMatches(), out.GetNextPageToken(), err
	})
	if err != nil {
		writeStatus(w, err)
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].GetKickoff(), items[j].GetKickoff()
		if a == nil || b == nil {
			return a != nil
		}
		return a.AsTime().Before(b.AsTime())
	})
	names := map[string]string{}
	teamName := func(id string) string {
		if name, ok := names[id]; ok {
			return name
		}
		name := "Unknown team"
		if out, err := s.c.Teams.GetTeam(ctx, &v1.GetTeamRequest{Id: id}); err == nil {
			name = out.GetTeam().GetName()
		} else {
			log.Printf("team %s lookup failed: %v", id, err)
		}
		names[id] = name
		return name
	}
	result := make([]map[string]any, 0, len(items))
	for _, m := range items {
		row := map[string]any{
			"id": m.GetId(), "tournament_id": m.GetTournamentId(),
			"home_team_id": m.GetHomeTeamId(), "away_team_id": m.GetAwayTeamId(),
			"home": teamName(m.GetHomeTeamId()), "away": teamName(m.GetAwayTeamId()),
			"venue": m.GetVenue(), "kickoff": timeOrNil(m.GetKickoff()), "status": m.GetStatus(), "official_id": m.GetOfficialId(),
			"live_home": nil, "live_away": nil, "result_home": nil, "result_away": nil,
		}
		score, err := s.c.Scores.GetScore(ctx, &v1.GetScoreRequest{MatchId: m.GetId()})
		switch sc := score.GetScore(); {
		case err != nil:
			log.Printf("score %s lookup failed: %v", m.GetId(), err)
		case sc.GetFinalized():
			row["result_home"], row["result_away"], row["status"] = sc.GetHomeScore(), sc.GetAwayScore(), "completed"
		case sc.GetUpdatedAt() != nil:
			row["live_home"], row["live_away"] = sc.GetHomeScore(), sc.GetAwayScore()
			if m.GetStatus() != "completed" {
				row["status"] = "live"
			}
		}
		result = append(result, row)
	}
	writeJSON(w, 200, result)
}

func (s *Server) createMatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TournamentID string `json:"tournament_id"`
		HomeTeamID   string `json:"home_team_id"`
		AwayTeamID   string `json:"away_team_id"`
		Venue        string `json:"venue"`
		Kickoff      string `json:"kickoff"`
		OfficialID   string `json:"official_id"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	kickoff, err := time.Parse(time.RFC3339, strings.TrimSpace(in.Kickoff))
	if err != nil {
		writeError(w, 400, "kickoff must be an RFC3339 timestamp")
		return
	}
	out, err := s.c.Matches.CreateMatch(r.Context(), &v1.CreateMatchRequest{TournamentId: in.TournamentID, HomeTeamId: in.HomeTeamID, AwayTeamId: in.AwayTeamID, Venue: in.Venue, Kickoff: timestamppb.New(kickoff)})
	if err != nil {
		writeStatus(w, err)
		return
	}
	id := out.GetMatch().GetId()
	if strings.TrimSpace(in.OfficialID) != "" {
		if _, err := s.c.Officials.AssignOfficial(r.Context(), &officialv1.AssignOfficialRequest{MatchId: id, OfficialUserId: in.OfficialID}); err != nil {
			writeStatus(w, err)
			return
		}
	}
	writeJSON(w, 201, map[string]string{"id": id, "status": out.GetMatch().GetStatus()})
}

func (s *Server) assignOfficial(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OfficialID string `json:"official_id"`
	}
	if !decode(w, r, &in, false) {
		return
	}
	matchID := r.PathValue("id")
	if _, err := s.c.Officials.AssignOfficial(r.Context(), &officialv1.AssignOfficialRequest{MatchId: matchID, OfficialUserId: in.OfficialID}); err != nil {
		writeStatus(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"match_id": matchID, "official_id": in.OfficialID})
}

type scoreBody struct {
	Home *int32 `json:"home"`
	Away *int32 `json:"away"`
}

func (s *Server) updateLive(w http.ResponseWriter, r *http.Request) {
	var in scoreBody
	if !decode(w, r, &in, false) {
		return
	}
	if in.Home == nil || in.Away == nil {
		writeError(w, 400, "home and away scores are required")
		return
	}
	out, err := s.c.Scores.UpdateLiveScore(r.Context(), &v1.UpdateLiveScoreRequest{MatchId: r.PathValue("id"), HomeScore: *in.Home, AwayScore: *in.Away})
	if err != nil {
		writeStatus(w, err)
		return
	}
	sc := out.GetScore()
	writeJSON(w, 200, map[string]any{"id": r.PathValue("id"), "live_home": sc.GetHomeScore(), "live_away": sc.GetAwayScore(), "status": "live"})
}

// finalize records the score the official submitted before finalising it. The monolith
// ignored the body and finalised whatever live score was saved last, so a score typed into
// the final-result form could be lost.
func (s *Server) finalize(w http.ResponseWriter, r *http.Request) {
	var in scoreBody
	if !decode(w, r, &in, true) {
		return
	}
	matchID := r.PathValue("id")
	if (in.Home == nil) != (in.Away == nil) {
		writeError(w, 400, "send both home and away scores, or neither")
		return
	}
	if in.Home != nil {
		if _, err := s.c.Scores.UpdateLiveScore(r.Context(), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: *in.Home, AwayScore: *in.Away}); err != nil {
			writeStatus(w, err)
			return
		}
	}
	out, err := s.c.Scores.FinalizeMatch(r.Context(), &v1.FinalizeMatchRequest{MatchId: matchID, Reason: "finalised from the web app"})
	if err != nil {
		writeStatus(w, err)
		return
	}
	sc := out.GetScore()
	writeJSON(w, 200, map[string]any{"id": matchID, "status": "completed", "home": sc.GetHomeScore(), "away": sc.GetAwayScore()})
}

// --- Standings

func (s *Server) listStandings(w http.ResponseWriter, r *http.Request) {
	tournamentID := r.URL.Query().Get("tournament_id")
	if tournamentID == "" {
		writeError(w, 400, "tournament_id is required")
		return
	}
	out, err := s.c.Standings.GetStandings(r.Context(), &standingsv1.GetStandingsRequest{TournamentId: tournamentID})
	if err != nil {
		writeStatus(w, err)
		return
	}
	result := make([]map[string]any, 0, len(out.GetStandings()))
	for _, row := range out.GetStandings() {
		result = append(result, map[string]any{
			"team_id": row.GetTeamId(), "team": row.GetTeamName(), "dzongkhag": row.GetDzongkhag(),
			"played": row.GetPlayed(), "won": row.GetWon(), "drawn": row.GetDrawn(), "lost": row.GetLost(),
			"goals_for": row.GetGoalsFor(), "goals_against": row.GetGoalsAgainst(), "goal_difference": row.GetGoalDifference(), "points": row.GetPoints(),
		})
	}
	writeJSON(w, 200, result)
}

// --- helpers

func timeOrNil(t *timestamppb.Timestamp) any {
	if t == nil {
		return nil
	}
	return t.AsTime()
}

// parseDateOrTime accepts the web form's date input (YYYY-MM-DD) or a full timestamp.
func parseDateOrTime(value string) (time.Time, bool) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}
