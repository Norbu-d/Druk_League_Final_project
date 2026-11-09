package natsadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/analytics/internal/domain"
)

type eventHandler struct {
	event domain.MatchFinalized
	err   error
}

func (h *eventHandler) RecordFinalized(_ context.Context, event domain.MatchFinalized) error {
	h.event = event
	return h.err
}

func TestHandler(t *testing.T) {
	valid := domain.MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222", TournamentID: "33333333-3333-4333-8333-333333333333", HomeTeamID: "44444444-4444-4444-8444-444444444444", AwayTeamID: "55555555-5555-4555-8555-555555555555", HomeScore: 1, FinalizedAt: time.Now()}
	validJSON, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		payload   []byte
		appErr    error
		permanent bool
	}{
		{"valid", validJSON, nil, false}, {"bad json", []byte("{"), nil, true},
		{"invalid event", []byte(`{"event_id":"bad"}`), nil, true},
		{"invalid from app", validJSON, domain.ErrInvalid, true},
		{"transient app error", validJSON, errors.New("retry"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &eventHandler{err: tt.appErr}
			err := Handler(h)(context.Background(), "ignored", tt.payload)
			if eventkit.IsPermanent(err) != tt.permanent {
				t.Fatalf("err=%v permanent=%v", err, eventkit.IsPermanent(err))
			}
			if tt.name == "valid" && h.event.EventID != valid.EventID {
				t.Fatalf("event=%+v", h.event)
			}
		})
	}
}

func TestConsumerConfiguration(t *testing.T) {
	cfg := Consumer(&eventHandler{})
	if cfg.Durable != "analytics-match-finalized-v1" || cfg.Subject != eventkit.SubjectMatchFinalized || cfg.Handler == nil {
		t.Fatalf("config=%+v", cfg)
	}
}
