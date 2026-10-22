package domain

import "time"

// Assignment records which match official runs a match. The Match service holds the
// authoritative official_id used for scoring; this service keeps the assignment record
// with its tournament so officials and admins can list assignments.
type Assignment struct {
	ID, MatchID, OfficialUserID, TournamentID, AssignedBy string
	AssignedAt                                            time.Time
}

// MatchRef is what the Match service returns after an assignment.
type MatchRef struct {
	ID, TournamentID, OfficialID string
}
