package domain

import (
	"testing"
	"time"
)

func TestMatchFinalizedValidation(t *testing.T) {
	valid := MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222", TournamentID: "33333333-3333-4333-8333-333333333333", HomeTeamID: "44444444-4444-4444-8444-444444444444", AwayTeamID: "55555555-5555-4555-8555-555555555555", HomeScore: 2, AwayScore: 1, FinalizedAt: time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	for _, mutate := range []func(*MatchFinalized){
		func(e *MatchFinalized) { e.EventID = "bad" },
		func(e *MatchFinalized) { e.MatchID = "bad" },
		func(e *MatchFinalized) { e.TournamentID = "bad" },
		func(e *MatchFinalized) { e.HomeTeamID = "bad" },
		func(e *MatchFinalized) { e.AwayTeamID = e.HomeTeamID },
		func(e *MatchFinalized) { e.HomeScore = -1 },
		func(e *MatchFinalized) { e.AwayScore = 100 },
		func(e *MatchFinalized) { e.FinalizedAt = time.Time{} },
	} {
		invalid := valid
		mutate(&invalid)
		if err := invalid.Validate(); err == nil {
			t.Errorf("invalid event accepted: %+v", invalid)
		}
	}
}
