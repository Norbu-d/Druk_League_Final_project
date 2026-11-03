package domain

import "time"

// Notification is an in-app message persisted for one user.
type Notification struct {
	ID, UserID, Channel, Subject, Body, EventID string
	CreatedAt                                   time.Time
}

// FinalizedMatch identifies the teams whose managers receive a notice.
type FinalizedMatch struct {
	EventID, MatchID, FinalizedBy, HomeTeamID, AwayTeamID string
	HomeScore, AwayScore                                  int32
}
