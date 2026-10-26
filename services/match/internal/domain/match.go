package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalid    = errors.New("invalid match request")
	ErrNotFound   = errors.New("match not found")
	ErrConflict   = errors.New("match scheduling conflict")
	ErrForbidden  = errors.New("operation is not permitted")
	ErrDependency = errors.New("dependent service request failed")
)

type Match struct {
	ID, TournamentID, HomeTeamID, AwayTeamID string
	Venue, Status, OfficialID                string
	Kickoff                                  time.Time
}

const (
	StatusScheduled = "scheduled"
	StatusCompleted = "completed"
)
