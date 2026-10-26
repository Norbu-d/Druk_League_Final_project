package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalid    = errors.New("invalid score request")
	ErrNotFound   = errors.New("score not found")
	ErrConflict   = errors.New("score is finalised")
	ErrForbidden  = errors.New("caller is not the assigned official")
	ErrDependency = errors.New("match lookup failed")
)

// MaxGoals bounds a submitted score, matching the monolith's validation.
const MaxGoals = 99

type Score struct {
	MatchID              string
	HomeScore, AwayScore int32
	Finalized            bool
	UpdatedAt            time.Time
}

type Match struct {
	ID, TournamentID, HomeTeamID, AwayTeamID, OfficialID, Status string
}

// MatchFinalized is the event payload Score writes to its outbox. FinalizedBy and Reason
// let Audit Log and Notification record who closed the result without calling back.
type MatchFinalized struct {
	EventID      string    `json:"event_id"`
	MatchID      string    `json:"match_id"`
	TournamentID string    `json:"tournament_id"`
	HomeTeamID   string    `json:"home_team_id"`
	AwayTeamID   string    `json:"away_team_id"`
	HomeScore    int32     `json:"home_score"`
	AwayScore    int32     `json:"away_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
	FinalizedBy  string    `json:"finalized_by,omitempty"`
	Reason       string    `json:"reason,omitempty"`
}

// OutboxEvent is one unpublished row of the transactional outbox.
type OutboxEvent struct {
	EventID string
	Payload []byte
}
