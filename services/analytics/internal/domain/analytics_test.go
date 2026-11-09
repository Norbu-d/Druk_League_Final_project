package domain

import (
	"errors"
	"testing"
	"time"
)

func TestMatchFinalizedNormalizeAndValidate(t *testing.T) {
	event := MatchFinalized{EventID: " 11111111-1111-4111-8111-111111111111 ", MatchID: "22222222-2222-4222-8222-222222222222", TournamentID: "33333333-3333-4333-8333-333333333333", HomeTeamID: "44444444-4444-4444-8444-444444444444", AwayTeamID: "55555555-5555-4555-8555-555555555555", HomeScore: 99, AwayScore: 0, FinalizedAt: time.Date(2026, 1, 1, 1, 0, 0, 0, time.FixedZone("test", 3600))}
	if err := event.NormalizeAndValidate(); err != nil {
		t.Fatal(err)
	}
	if event.EventID != "11111111-1111-4111-8111-111111111111" || event.FinalizedAt.Location() != time.UTC {
		t.Fatalf("not normalized: %+v", event)
	}
}

func TestMatchFinalizedRejectsInvalidFields(t *testing.T) {
	base := MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222", TournamentID: "33333333-3333-4333-8333-333333333333", HomeTeamID: "44444444-4444-4444-8444-444444444444", AwayTeamID: "55555555-5555-4555-8555-555555555555", HomeScore: 1, AwayScore: 0, FinalizedAt: time.Now()}
	tests := []struct {
		name   string
		mutate func(*MatchFinalized)
	}{
		{"bad event id", func(e *MatchFinalized) { e.EventID = "bad" }}, {"bad match id", func(e *MatchFinalized) { e.MatchID = "bad" }},
		{"bad tournament id", func(e *MatchFinalized) { e.TournamentID = "bad" }}, {"bad home team", func(e *MatchFinalized) { e.HomeTeamID = "bad" }},
		{"bad away team", func(e *MatchFinalized) { e.AwayTeamID = "bad" }}, {"same teams", func(e *MatchFinalized) { e.AwayTeamID = e.HomeTeamID }},
		{"negative home score", func(e *MatchFinalized) { e.HomeScore = -1 }}, {"high home score", func(e *MatchFinalized) { e.HomeScore = 100 }},
		{"negative away score", func(e *MatchFinalized) { e.AwayScore = -1 }}, {"high away score", func(e *MatchFinalized) { e.AwayScore = 100 }},
		{"missing time", func(e *MatchFinalized) { e.FinalizedAt = time.Time{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := base
			tt.mutate(&event)
			if err := event.NormalizeAndValidate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
