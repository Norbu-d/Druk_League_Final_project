package domain

import "errors"

var (
	ErrInvalid    = errors.New("invalid finalized match event")
	ErrDependency = errors.New("team lookup failed")
)

type MatchFinalized struct {
	EventID      string `json:"event_id"`
	MatchID      string `json:"match_id"`
	TournamentID string `json:"tournament_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeScore    int32  `json:"home_score"`
	AwayScore    int32  `json:"away_score"`
}

type Team struct{ ID, Name, Dzongkhag, TournamentID string }
type Standing struct {
	TeamID, TeamName, Dzongkhag                    string
	Played, Won, Drawn, Lost                       int32
	GoalsFor, GoalsAgainst, GoalDifference, Points int32
}

// Outcome converts one side's goals into a standings delta: 3 points for a win, 1 for a draw.
func Outcome(goalsFor, goalsAgainst int32) (won, drawn, lost, points int32) {
	switch {
	case goalsFor > goalsAgainst:
		return 1, 0, 0, 3
	case goalsFor == goalsAgainst:
		return 0, 1, 0, 1
	default:
		return 0, 0, 1, 0
	}
}
