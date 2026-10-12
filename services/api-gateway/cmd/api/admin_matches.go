package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (s *server) createUser(w http.ResponseWriter, r *http.Request, p principal) {
	if p.Role != "super_admin" {
		writeError(w, 403, "Only a super admin can create user accounts")
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || len(in.Password) < 10 {
		writeError(w, 400, "username is required and password must be at least 10 characters")
		return
	}
	if in.Role != "match_official" && in.Role != "viewer" {
		writeError(w, 400, "role must be match_official or viewer")
		return
	}
	var id string
	e := s.db.QueryRowContext(r.Context(), `INSERT INTO users(username,password_hash,role) VALUES($1,$2,$3) RETURNING id::text`, in.Username, hashPassword(in.Password), in.Role).Scan(&id)
	if e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "username": in.Username, "role": in.Role})
}

func (s *server) createMatch(w http.ResponseWriter, r *http.Request, p principal) {
	if !canManage(p) {
		writeError(w, 403, "Only administrators can create fixtures")
		return
	}
	var in struct {
		TournamentID string `json:"tournament_id"`
		HomeTeamID   string `json:"home_team_id"`
		AwayTeamID   string `json:"away_team_id"`
		Venue        string `json:"venue"`
		Kickoff      string `json:"kickoff"`
		OfficialID   string `json:"official_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.TournamentID == "" || in.HomeTeamID == "" || in.AwayTeamID == "" || in.Kickoff == "" {
		writeError(w, 400, "tournament_id, home_team_id, away_team_id, and kickoff are required")
		return
	}
	start, e := time.Parse(time.RFC3339, in.Kickoff)
	if e != nil {
		writeError(w, 400, "kickoff must be an RFC3339 timestamp")
		return
	}
	if in.HomeTeamID == in.AwayTeamID {
		writeError(w, 400, "A team cannot play itself")
		return
	}

	if in.OfficialID != "" {
		var role string
		e = s.db.QueryRowContext(r.Context(), `SELECT role FROM users WHERE id=$1 AND role='match_official'`, in.OfficialID).Scan(&role)
		if e != nil {
			writeError(w, 400, "official_id must refer to a match official")
			return
		}
	}
	var id string
	e = s.db.QueryRowContext(r.Context(), `WITH valid_teams AS (SELECT count(*) AS n FROM teams WHERE tournament_id=$1 AND id IN ($2::uuid,$3::uuid)), clashes AS (SELECT count(*) AS n FROM matches WHERE kickoff=$4 AND (home_team_id IN ($2::uuid,$3::uuid) OR away_team_id IN ($2::uuid,$3::uuid) OR ($5<>'' AND venue=$5))) INSERT INTO matches(tournament_id,home_team_id,away_team_id,venue,kickoff,official_id) SELECT $1,$2,$3,$5,$4,NULLIF($6,'')::uuid WHERE (SELECT n FROM valid_teams)=2 AND (SELECT n FROM clashes)=0 RETURNING id::text`, in.TournamentID, in.HomeTeamID, in.AwayTeamID, start, in.Venue, in.OfficialID).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		writeError(w, 409, "Teams must belong to this tournament and the time slot must have no team or venue clash")
		return
	}
	if e != nil {
		serverError(w, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "status": "scheduled"})
}

func (s *server) listUsers(w http.ResponseWriter, r *http.Request, p principal) {
	if p.Role != "super_admin" {
		writeError(w, 403, "Only a super admin can list accounts")
		return
	}
	role := r.URL.Query().Get("role")
	query := `SELECT id::text,username,role FROM users`
	args := []any{}
	if role != "" {
		query += ` WHERE role=$1`
		args = append(args, role)
	}
	query += ` ORDER BY username`
	rows, e := s.db.QueryContext(r.Context(), query, args...)
	if e != nil {
		serverError(w, e)
		return
	}
	defer rows.Close()
	result := []map[string]string{}
	for rows.Next() {
		var id, user, accountRole string
		if e = rows.Scan(&id, &user, &accountRole); e != nil {
			serverError(w, e)
			return
		}
		result = append(result, map[string]string{"id": id, "username": user, "role": accountRole})
	}
	writeJSON(w, 200, result)
}

func (s *server) assignOfficial(w http.ResponseWriter, r *http.Request, p principal, matchID string) {
	if !canManage(p) {
		writeError(w, 403, "Only administrators can assign officials")
		return
	}
	var in struct {
		OfficialID string `json:"official_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.OfficialID == "" {
		writeError(w, 400, "official_id is required")
		return
	}
	var role string
	e := s.db.QueryRowContext(r.Context(), `SELECT role FROM users WHERE id=$1`, in.OfficialID).Scan(&role)
	if e != nil || role != "match_official" {
		writeError(w, 400, "official_id must refer to a match official")
		return
	}

	res, e := s.db.ExecContext(r.Context(), `UPDATE matches SET official_id=$1 WHERE id=$2 AND finalized_at IS NULL`, in.OfficialID, matchID)
	if e != nil {
		serverError(w, e)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 409, "Match is finalized or does not exist")
		return
	}
	writeJSON(w, 200, map[string]string{"match_id": matchID, "official_id": in.OfficialID})
}
