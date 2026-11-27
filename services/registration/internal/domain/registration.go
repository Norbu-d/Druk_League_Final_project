package domain

import "time"

// Registration statuses. A registration starts pending and is reviewed exactly once.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// TournamentOpen is the tournament status that accepts new registrations.
const TournamentOpen = "registration_open"

// Registration is a team's request to enter a tournament.
type Registration struct {
	ID, TournamentID, TeamID, Status       string
	RegisteredBy, ReviewedBy, ReviewReason string
	SubmittedAt                            time.Time
}

// TeamRef is what the Team service tells us about a team.
type TeamRef struct{ ID, ManagerID string }

// TournamentRef is what the Tournament service tells us about a tournament.
type TournamentRef struct{ ID, Status string }

// ValidStatus reports whether s is a known registration status.
func ValidStatus(s string) bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	}
	return false
}
