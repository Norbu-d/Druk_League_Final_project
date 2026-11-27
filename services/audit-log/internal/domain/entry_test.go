package domain

import (
	"strings"
	"testing"
	"time"
)

const (
	eventID = "11111111-1111-4111-8111-111111111111"
	matchID = "22222222-2222-4222-8222-222222222222"
)

func TestAdminActionValidate(t *testing.T) {
	valid := AdminAction{EventID: eventID, ActorID: "admin", Action: "team.updated", ResourceType: "team", ResourceID: "team-1", OccurredAt: time.Now()}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	for name, mutate := range map[string]func(*AdminAction){
		"event id":      func(e *AdminAction) { e.EventID = "bad" },
		"actor":         func(e *AdminAction) { e.ActorID = " " },
		"action":        func(e *AdminAction) { e.Action = " " },
		"resource type": func(e *AdminAction) { e.ResourceType = " " },
		"resource id":   func(e *AdminAction) { e.ResourceID = " " },
		"timestamp":     func(e *AdminAction) { e.OccurredAt = time.Time{} },
	} {
		t.Run(name, func(t *testing.T) {
			e := valid
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatal("expected invalid event")
			}
		})
	}
	for name, mutate := range map[string]func(*AdminAction){
		"actor":         func(e *AdminAction) { e.ActorID = strings.Repeat("a", 201) },
		"action":        func(e *AdminAction) { e.Action = strings.Repeat("a", 201) },
		"resource type": func(e *AdminAction) { e.ResourceType = strings.Repeat("a", 101) },
		"resource id":   func(e *AdminAction) { e.ResourceID = strings.Repeat("a", 201) },
		"reason":        func(e *AdminAction) { e.Reason = strings.Repeat("a", 2001) },
	} {
		t.Run(name+" limit", func(t *testing.T) {
			e := valid
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatal("expected over-limit event")
			}
		})
	}
}

func TestMatchFinalizedValidate(t *testing.T) {
	valid := MatchFinalized{EventID: eventID, MatchID: matchID, FinalizedBy: "official", FinalizedAt: time.Now()}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	for name, mutate := range map[string]func(*MatchFinalized){
		"event id":     func(e *MatchFinalized) { e.EventID = "bad" },
		"match id":     func(e *MatchFinalized) { e.MatchID = "bad" },
		"timestamp":    func(e *MatchFinalized) { e.FinalizedAt = time.Time{} },
		"actor limit":  func(e *MatchFinalized) { e.FinalizedBy = strings.Repeat("a", 201) },
		"reason limit": func(e *MatchFinalized) { e.Reason = strings.Repeat("a", 2001) },
	} {
		t.Run(name, func(t *testing.T) {
			e := valid
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatal("expected invalid event")
			}
		})
	}
}

func TestEntryValidate(t *testing.T) {
	valid := Entry{ActorID: "admin", Action: "team.updated", ResourceType: "team", ResourceID: "team-1", OccurredAt: time.Now()}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid entry rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Entry){
		"actor":               func(e *Entry) { e.ActorID = " " },
		"action":              func(e *Entry) { e.Action = " " },
		"resource type":       func(e *Entry) { e.ResourceType = " " },
		"resource id":         func(e *Entry) { e.ResourceID = " " },
		"timestamp":           func(e *Entry) { e.OccurredAt = time.Time{} },
		"actor limit":         func(e *Entry) { e.ActorID = strings.Repeat("a", 201) },
		"action limit":        func(e *Entry) { e.Action = strings.Repeat("a", 201) },
		"resource type limit": func(e *Entry) { e.ResourceType = strings.Repeat("a", 101) },
		"resource id limit":   func(e *Entry) { e.ResourceID = strings.Repeat("a", 201) },
		"reason limit":        func(e *Entry) { e.Reason = strings.Repeat("a", 2001) },
	} {
		t.Run(name, func(t *testing.T) {
			e := valid
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatal("expected invalid entry")
			}
		})
	}
}
