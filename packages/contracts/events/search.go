package events

import "time"

const (
	SearchTeamSubject       = "drukleague.search.team"
	SearchPlayerSubject     = "drukleague.search.player"
	SearchTournamentSubject = "drukleague.search.tournament"

	SearchOperationUpsert = "upsert"
	SearchOperationDelete = "delete"
)

type SearchDocumentFields struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	SearchText string `json:"search_text"`
}

type SearchIndexEvent struct {
	EventID    string               `json:"event_id"`
	EntityType string               `json:"entity_type"`
	EntityID   string               `json:"entity_id"`
	Version    int64                `json:"version"`
	Operation  string               `json:"operation"`
	OccurredAt time.Time            `json:"occurred_at"`
	Fields     SearchDocumentFields `json:"fields"`
}

func SearchSubject(entityType string) string {
	switch entityType {
	case "team":
		return SearchTeamSubject
	case "player":
		return SearchPlayerSubject
	case "tournament":
		return SearchTournamentSubject
	default:
		return ""
	}
}
