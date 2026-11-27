package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const adminResultCorrectedSubject = "drukleague.admin.match.result.corrected"

type adminActionEvent struct {
	EventID      string    `json:"event_id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Reason       string    `json:"reason"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type resultContribution struct {
	won, drawn, lost int
	goalsFor         int
	goalsAgainst     int
	points           int
}

func contribution(goalsFor, goalsAgainst int) resultContribution {
	result := resultContribution{goalsFor: goalsFor, goalsAgainst: goalsAgainst}
	switch {
	case goalsFor > goalsAgainst:
		result.won, result.points = 1, 3
	case goalsFor < goalsAgainst:
		result.lost = 1
	default:
		result.drawn, result.points = 1, 1
	}
	return result
}

func (s *server) correctResult(w http.ResponseWriter, r *http.Request, p principal, matchID string) {
	if !canManage(p) {
		writeError(w, http.StatusForbidden, "Only administrators can correct match results")
		return
	}

	var in struct {
		Home   int    `json:"home_score"`
		Away   int    `json:"away_score"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Home < 0 || in.Home > 99 || in.Away < 0 || in.Away > 99 {
		writeError(w, http.StatusBadRequest, "Scores must be between 0 and 99")
		return
	}
	if in.Reason == "" || len(in.Reason) > 2000 {
		writeError(w, http.StatusBadRequest, "A correction reason of at most 2000 characters is required")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()

	var tournamentID, homeTeamID, awayTeamID string
	var oldHome, oldAway sql.NullInt64
	var finalizedAt sql.NullTime
	err = tx.QueryRowContext(r.Context(), `
		SELECT tournament_id::text, home_team_id::text, away_team_id::text,
		       result_home, result_away, finalized_at
		FROM matches WHERE id=$1::uuid FOR UPDATE`, matchID,
	).Scan(&tournamentID, &homeTeamID, &awayTeamID, &oldHome, &oldAway, &finalizedAt)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Match not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if !finalizedAt.Valid || !oldHome.Valid || !oldAway.Valid {
		writeError(w, http.StatusConflict, "Only a finalized match result can be corrected")
		return
	}
	if int(oldHome.Int64) == in.Home && int(oldAway.Int64) == in.Away {
		writeError(w, http.StatusBadRequest, "The corrected score must differ from the current result")
		return
	}

	// The database trigger keeps finalized scores immutable. This transaction-local flag
	// opens the exception only for this authenticated, admin-only correction operation.
	if err = tx.QueryRowContext(r.Context(), `SELECT set_config('drukleague.allow_result_correction','on',true)`).Scan(new(string)); err != nil {
		serverError(w, err)
		return
	}
	correctedAt := time.Now().UTC()
	if _, err = tx.ExecContext(r.Context(), `
		UPDATE matches SET result_home=$1, result_away=$2, finalized_at=$3, status='completed'
		WHERE id=$4::uuid`, in.Home, in.Away, correctedAt, matchID,
	); err != nil {
		serverError(w, err)
		return
	}

	oldHomeStats := contribution(int(oldHome.Int64), int(oldAway.Int64))
	newHomeStats := contribution(in.Home, in.Away)
	oldAwayStats := contribution(int(oldAway.Int64), int(oldHome.Int64))
	newAwayStats := contribution(in.Away, in.Home)
	for _, update := range []struct {
		teamID string
		old    resultContribution
		new    resultContribution
	}{
		{teamID: homeTeamID, old: oldHomeStats, new: newHomeStats},
		{teamID: awayTeamID, old: oldAwayStats, new: newAwayStats},
	} {
		result, updateErr := tx.ExecContext(r.Context(), `
			UPDATE standings
			SET won=won+$1, drawn=drawn+$2, lost=lost+$3,
			    goals_for=goals_for+$4, goals_against=goals_against+$5, points=points+$6
			WHERE tournament_id=$7::uuid AND team_id=$8::uuid`,
			update.new.won-update.old.won,
			update.new.drawn-update.old.drawn,
			update.new.lost-update.old.lost,
			update.new.goalsFor-update.old.goalsFor,
			update.new.goalsAgainst-update.old.goalsAgainst,
			update.new.points-update.old.points,
			tournamentID, update.teamID,
		)
		if updateErr != nil {
			serverError(w, updateErr)
			return
		}
		if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
			if rowsErr != nil {
				serverError(w, rowsErr)
			} else {
				serverError(w, errors.New("standings row missing for finalized match"))
			}
			return
		}
	}

	var eventID string
	if err = tx.QueryRowContext(r.Context(), `SELECT gen_random_uuid()::text`).Scan(&eventID); err != nil {
		serverError(w, err)
		return
	}
	if _, err = tx.ExecContext(r.Context(), `
		INSERT INTO match_result_corrections
		    (event_id, match_id, actor_id, old_home_score, old_away_score, new_home_score, new_away_score, reason, corrected_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)`,
		eventID, matchID, p.ID, oldHome.Int64, oldAway.Int64, in.Home, in.Away, in.Reason, correctedAt,
	); err != nil {
		serverError(w, err)
		return
	}
	payload, err := json.Marshal(adminActionEvent{
		EventID: eventID, ActorID: p.ID, Action: "match.result.corrected", ResourceType: "match",
		ResourceID: matchID, Reason: in.Reason, OccurredAt: correctedAt,
	})
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err = tx.ExecContext(r.Context(), `
		INSERT INTO event_outbox(subject,payload) VALUES($1,$2::jsonb)`,
		adminResultCorrectedSubject, string(payload),
	); err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              matchID,
		"status":          "completed",
		"previous_result": map[string]int{"home": int(oldHome.Int64), "away": int(oldAway.Int64)},
		"result":          map[string]int{"home": in.Home, "away": in.Away},
		"corrected_at":    correctedAt,
	})
}
