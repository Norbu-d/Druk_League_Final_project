package natsadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

type recorderStub struct {
	admin                  domain.AdminAction
	match                  domain.MatchFinalized
	adminCalls, matchCalls int
	err                    error
}

func (r *recorderStub) RecordAdminAction(_ context.Context, event domain.AdminAction) error {
	r.adminCalls++
	r.admin = event
	return r.err
}
func (r *recorderStub) RecordMatchFinalized(_ context.Context, event domain.MatchFinalized) error {
	r.matchCalls++
	r.match = event
	return r.err
}

func TestConsumerConfigs(t *testing.T) {
	match := MatchFinalizedConsumer(&recorderStub{})
	if match.Durable != "audit-log-match-finalized-v1" || match.Subject != eventkit.SubjectMatchFinalized || match.MaxDeliver != 5 || match.Backoff != 2*time.Second || match.Handler == nil {
		t.Fatalf("MatchFinalized config = %+v", match)
	}
	admin := AdminActionsConsumer(&recorderStub{})
	if admin.Durable != "audit-log-admin-actions-v1" || admin.Subject != eventkit.SubjectAdminActions || admin.MaxDeliver != 5 || admin.Backoff != 2*time.Second || admin.Handler == nil {
		t.Fatalf("admin config = %+v", admin)
	}
}

func TestMatchFinalizedHandler(t *testing.T) {
	service := &recorderStub{}
	config := MatchFinalizedConsumer(service)
	payload := []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","match_id":"22222222-2222-4222-8222-222222222222","finalized_by":"referee","finalized_at":"2026-11-06T12:00:00Z"}`)
	if err := config.Handler(context.Background(), eventkit.SubjectMatchFinalized, payload); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if service.matchCalls != 1 || service.match.MatchID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("recorded match = %+v calls=%d", service.match, service.matchCalls)
	}

	for name, tc := range map[string]struct {
		subject string
		body    []byte
	}{
		"wrong subject":  {"drukleague.admin.user.updated", payload},
		"malformed JSON": {eventkit.SubjectMatchFinalized, []byte("{")},
		"invalid event":  {eventkit.SubjectMatchFinalized, []byte(`{"event_id":"bad"}`)},
	} {
		t.Run(name, func(t *testing.T) {
			err := config.Handler(context.Background(), tc.subject, tc.body)
			if err == nil || !eventkit.IsPermanent(err) {
				t.Fatalf("error=%v, want permanent", err)
			}
		})
	}
	service.err = errors.New("database unavailable")
	if err := config.Handler(context.Background(), eventkit.SubjectMatchFinalized, payload); err == nil || eventkit.IsPermanent(err) {
		t.Fatalf("recorder error=%v, want retryable error", err)
	}
}

func TestAdminActionHandler(t *testing.T) {
	service := &recorderStub{}
	config := AdminActionsConsumer(service)
	payload := []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","actor_id":"admin","action":"team.updated","resource_type":"team","resource_id":"team-1","occurred_at":"2026-11-06T12:00:00Z"}`)
	if err := config.Handler(context.Background(), "drukleague.admin.team.updated", payload); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if service.adminCalls != 1 || service.admin.Action != "team.updated" {
		t.Fatalf("recorded admin event = %+v calls=%d", service.admin, service.adminCalls)
	}
	for name, tc := range map[string]struct {
		subject string
		body    []byte
	}{
		"wrong subject":  {eventkit.SubjectMatchFinalized, payload},
		"prefix only":    {"drukleague.admin.", payload},
		"malformed JSON": {"drukleague.admin.user.updated", []byte("{")},
		"invalid event":  {"drukleague.admin.user.updated", []byte(`{"event_id":"bad"}`)},
	} {
		t.Run(name, func(t *testing.T) {
			err := config.Handler(context.Background(), tc.subject, tc.body)
			if err == nil || !eventkit.IsPermanent(err) {
				t.Fatalf("error=%v, want permanent", err)
			}
		})
	}
	service.err = errors.New("database unavailable")
	if err := config.Handler(context.Background(), "drukleague.admin.user.updated", payload); err == nil || eventkit.IsPermanent(err) {
		t.Fatalf("recorder error=%v, want retryable error", err)
	}
}

func TestSubjectMatchesAdmin(t *testing.T) {
	for subject, want := range map[string]bool{"drukleague.admin.user.updated": true, "drukleague.admin.": false, "drukleague.match.finalized": false, "": false} {
		if got := subjectMatchesAdmin(subject); got != want {
			t.Errorf("subjectMatchesAdmin(%q) = %v, want %v", subject, got, want)
		}
	}
}
