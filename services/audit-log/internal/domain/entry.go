package domain

import (
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

type Entry struct {
	ID, EventID, ActorID, Action, ResourceType, ResourceID, Reason string
	OccurredAt                                                     time.Time
}

type AdminAction struct {
	EventID      string    `json:"event_id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Reason       string    `json:"reason"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type MatchFinalized struct {
	EventID     string    `json:"event_id"`
	MatchID     string    `json:"match_id"`
	FinalizedBy string    `json:"finalized_by"`
	Reason      string    `json:"reason"`
	FinalizedAt time.Time `json:"finalized_at"`
}

func (e AdminAction) Validate() error {
	e.EventID = strings.TrimSpace(e.EventID)
	e.ActorID = strings.TrimSpace(e.ActorID)
	e.Action = strings.TrimSpace(e.Action)
	e.ResourceType = strings.TrimSpace(e.ResourceType)
	e.ResourceID = strings.TrimSpace(e.ResourceID)
	if !core.IsUUID(e.EventID) || e.ActorID == "" || e.Action == "" || e.ResourceType == "" || e.ResourceID == "" || e.OccurredAt.IsZero() {
		return core.Invalid("admin action event is malformed")
	}
	if len(e.ActorID) > 200 || len(e.Action) > 200 || len(e.ResourceType) > 100 || len(e.ResourceID) > 200 || len(e.Reason) > 2000 {
		return core.Invalid("admin action event fields exceed their limits")
	}
	return nil
}

func (e MatchFinalized) Validate() error {
	e.EventID = strings.TrimSpace(e.EventID)
	e.MatchID = strings.TrimSpace(e.MatchID)
	e.FinalizedBy = strings.TrimSpace(e.FinalizedBy)
	if !core.IsUUID(e.EventID) || !core.IsUUID(e.MatchID) || e.FinalizedAt.IsZero() {
		return core.Invalid("MatchFinalized event is malformed")
	}
	if len(e.FinalizedBy) > 200 || len(e.Reason) > 2000 {
		return core.Invalid("MatchFinalized fields exceed their limits")
	}
	return nil
}

func (e Entry) Validate() error {
	e.ActorID = strings.TrimSpace(e.ActorID)
	e.Action = strings.TrimSpace(e.Action)
	e.ResourceType = strings.TrimSpace(e.ResourceType)
	e.ResourceID = strings.TrimSpace(e.ResourceID)
	if e.ActorID == "" || e.Action == "" || e.ResourceType == "" || e.ResourceID == "" || e.OccurredAt.IsZero() {
		return core.Invalid("actor, action, resource type, resource id and occurrence time are required")
	}
	if len(e.ActorID) > 200 || len(e.Action) > 200 || len(e.ResourceType) > 100 || len(e.ResourceID) > 200 || len(e.Reason) > 2000 {
		return core.Invalid("audit entry fields exceed their limits")
	}
	return nil
}
