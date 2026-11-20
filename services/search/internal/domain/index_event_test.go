package domain

import (
	"errors"
	"testing"
)

const testTeamID = "11111111-1111-4111-8111-111111111111"
const testEventID = "22222222-2222-4222-8222-222222222222"

func TestDecodeIndexEventUpsertNormalizesFields(t *testing.T) {
	event, err := DecodeIndexEvent([]byte(`{"event_id":"22222222-2222-4222-8222-222222222222","entity_type":"team","entity_id":"11111111-1111-4111-8111-111111111111","version":1,"operation":"upsert","occurred_at":"2026-11-20T00:00:00Z","fields":{"title":"  Thimphu FC ","subtitle":"  Thimphu "}}`), EntityTeam)
	if err != nil {
		t.Fatal(err)
	}
	if event.Fields.Title != "Thimphu FC" || event.Fields.Subtitle != "Thimphu" || event.Fields.SearchText != "Thimphu FC Thimphu" {
		t.Fatalf("fields = %+v", event.Fields)
	}
}

func TestDecodeIndexEventDeleteDoesNotRequireFields(t *testing.T) {
	event, err := DecodeIndexEvent([]byte(`{"event_id":"22222222-2222-4222-8222-222222222222","entity_type":"team","entity_id":"11111111-1111-4111-8111-111111111111","version":2,"operation":"delete","occurred_at":"2026-11-20T00:00:00Z"}`), EntityTeam)
	if err != nil || event.Operation != OperationDelete {
		t.Fatalf("event=%+v err=%v", event, err)
	}
}

func TestDecodeIndexEventRejectsInvalidEnvelopes(t *testing.T) {
	valid := `{"event_id":"22222222-2222-4222-8222-222222222222","entity_type":"team","entity_id":"11111111-1111-4111-8111-111111111111","version":1,"operation":"upsert","occurred_at":"2026-11-20T00:00:00Z","fields":{"title":"Team"}}`
	cases := []struct {
		name    string
		payload string
		entity  string
	}{
		{"malformed json", `{`, EntityTeam},
		{"wrong expected entity", valid, EntityPlayer},
		{"unsupported entity", `{"event_id":"` + testEventID + `","entity_type":"venue","entity_id":"` + testTeamID + `","version":1,"operation":"delete","occurred_at":"2026-11-20T00:00:00Z"}`, EntityTeam},
		{"invalid event id", `{"event_id":"bad","entity_type":"team","entity_id":"` + testTeamID + `","version":1,"operation":"delete","occurred_at":"2026-11-20T00:00:00Z"}`, EntityTeam},
		{"invalid entity id", `{"event_id":"` + testEventID + `","entity_type":"team","entity_id":"bad","version":1,"operation":"delete","occurred_at":"2026-11-20T00:00:00Z"}`, EntityTeam},
		{"nonpositive version", `{"event_id":"` + testEventID + `","entity_type":"team","entity_id":"` + testTeamID + `","version":0,"operation":"delete","occurred_at":"2026-11-20T00:00:00Z"}`, EntityTeam},
		{"missing timestamp", `{"event_id":"` + testEventID + `","entity_type":"team","entity_id":"` + testTeamID + `","version":1,"operation":"delete"}`, EntityTeam},
		{"missing upsert title", `{"event_id":"` + testEventID + `","entity_type":"team","entity_id":"` + testTeamID + `","version":1,"operation":"upsert","occurred_at":"2026-11-20T00:00:00Z","fields":{"subtitle":"x"}}`, EntityTeam},
		{"unknown operation", `{"event_id":"` + testEventID + `","entity_type":"team","entity_id":"` + testTeamID + `","version":1,"operation":"create","occurred_at":"2026-11-20T00:00:00Z"}`, EntityTeam},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeIndexEvent([]byte(tc.payload), tc.entity); !errors.Is(err, ErrInvalidIndexEvent) && tc.name != "malformed json" {
				t.Fatalf("error = %v, want ErrInvalidIndexEvent", err)
			} else if err == nil {
				t.Fatal("expected event validation error")
			}
		})
	}
}
