package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

const (
	EntityTeam       = "team"
	EntityPlayer     = "player"
	EntityTournament = "tournament"

	OperationUpsert = "upsert"
	OperationDelete = "delete"
)

var ErrInvalidIndexEvent = errors.New("invalid search index event")

type DocumentFields struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	SearchText string `json:"search_text"`
}

type IndexEvent struct {
	EventID    string         `json:"event_id"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Version    int64          `json:"version"`
	Operation  string         `json:"operation"`
	OccurredAt time.Time      `json:"occurred_at"`
	Fields     DocumentFields `json:"fields"`
}

func DecodeIndexEvent(payload []byte, expectedEntity string) (IndexEvent, error) {
	var event IndexEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return IndexEvent{}, err
	}
	if !core.IsUUID(event.EventID) || !core.IsUUID(event.EntityID) || event.Version < 1 || event.OccurredAt.IsZero() || event.EntityType != expectedEntity {
		return IndexEvent{}, ErrInvalidIndexEvent
	}
	switch event.EntityType {
	case EntityTeam, EntityPlayer, EntityTournament:
	default:
		return IndexEvent{}, ErrInvalidIndexEvent
	}
	switch event.Operation {
	case OperationUpsert:
		event.Fields.Title = strings.TrimSpace(event.Fields.Title)
		event.Fields.Subtitle = strings.TrimSpace(event.Fields.Subtitle)
		event.Fields.SearchText = strings.TrimSpace(event.Fields.SearchText)
		if event.Fields.Title == "" {
			return IndexEvent{}, ErrInvalidIndexEvent
		}
		if event.Fields.SearchText == "" {
			event.Fields.SearchText = strings.TrimSpace(event.Fields.Title + " " + event.Fields.Subtitle)
		}
	case OperationDelete:
	default:
		return IndexEvent{}, ErrInvalidIndexEvent
	}
	return event, nil
}
