package domain

import (
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

type Notification struct {
	ID, EventID, UserID, Channel, Subject, Body, Status string
	CreatedAt                                           time.Time
}

type Team struct {
	ID, Name, ShortCode, Dzongkhag, TournamentID, ManagerID string
}

type MatchFinalized struct {
	EventID      string    `json:"event_id"`
	MatchID      string    `json:"match_id"`
	TournamentID string    `json:"tournament_id"`
	HomeTeamID   string    `json:"home_team_id"`
	AwayTeamID   string    `json:"away_team_id"`
	HomeScore    int32     `json:"home_score"`
	AwayScore    int32     `json:"away_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
}

func (e MatchFinalized) Validate() error {
	e.EventID = strings.TrimSpace(e.EventID)
	e.MatchID = strings.TrimSpace(e.MatchID)
	e.TournamentID = strings.TrimSpace(e.TournamentID)
	e.HomeTeamID = strings.TrimSpace(e.HomeTeamID)
	e.AwayTeamID = strings.TrimSpace(e.AwayTeamID)
	if !core.IsUUID(e.EventID) || !core.IsUUID(e.MatchID) || !core.IsUUID(e.TournamentID) ||
		!core.IsUUID(e.HomeTeamID) || !core.IsUUID(e.AwayTeamID) || e.HomeTeamID == e.AwayTeamID ||
		e.HomeScore < 0 || e.HomeScore > 99 || e.AwayScore < 0 || e.AwayScore > 99 || e.FinalizedAt.IsZero() {
		return core.Invalid("MatchFinalized event is malformed")
	}
	return nil
}
