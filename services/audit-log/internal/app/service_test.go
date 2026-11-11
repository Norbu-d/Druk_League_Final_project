package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

type repositoryStub struct {
	recorded           domain.Entry
	entries            []domain.Entry
	more               bool
	recordErr, listErr error
	listArgs           []any
}

func (r *repositoryStub) Record(_ context.Context, item domain.Entry) (domain.Entry, error) {
	r.recorded = item
	if r.recordErr != nil {
		return domain.Entry{}, r.recordErr
	}
	item.ID = "stored-id"
	return item, nil
}
func (r *repositoryStub) List(_ context.Context, resourceType, resourceID string, limit, offset int) ([]domain.Entry, bool, error) {
	r.listArgs = []any{resourceType, resourceID, limit, offset}
	return r.entries, r.more, r.listErr
}

func TestRecordAuthorizationNormalizationAndDefaultTime(t *testing.T) {
	repo := &repositoryStub{}
	svc := New(repo)
	if _, err := svc.Record(context.Background(), Caller{ID: "x", Role: core.RoleTeamManager}, domain.Entry{}); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("unauthorized record error = %v", err)
	}

	item, err := svc.Record(context.Background(), Caller{ID: " admin ", Role: core.RoleSuperAdmin}, domain.Entry{Action: " team.updated ", ResourceType: " team ", ResourceID: " t-1 ", Reason: " corrected ", OccurredAt: time.Time{}})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if item.ID != "stored-id" || repo.recorded.ActorID != "admin" || repo.recorded.Action != "team.updated" || repo.recorded.ResourceType != "team" || repo.recorded.ResourceID != "t-1" || repo.recorded.Reason != "corrected" || repo.recorded.OccurredAt.IsZero() || repo.recorded.OccurredAt.Location() != time.UTC {
		t.Fatalf("normalized record = %+v", repo.recorded)
	}

	fixed := time.Date(2026, 11, 6, 12, 0, 0, 0, time.FixedZone("test", 3600))
	_, err = svc.Record(context.Background(), Caller{ID: "svc", Role: core.RoleService}, domain.Entry{Action: " a ", ResourceType: " r ", ResourceID: " 1 ", OccurredAt: fixed})
	if err != nil {
		t.Fatalf("service record: %v", err)
	}
	if repo.recorded.OccurredAt != fixed || repo.recorded.ActorID != "svc" {
		t.Fatalf("service record = %+v", repo.recorded)
	}

	repo.recordErr = errors.New("storage failed")
	if _, err := svc.Record(context.Background(), Caller{ID: "admin", Role: core.RoleSuperAdmin}, domain.Entry{Action: "a", ResourceType: "r", ResourceID: "1", OccurredAt: time.Now()}); !errors.Is(err, repo.recordErr) {
		t.Fatalf("repository error = %v", err)
	}
}

func TestRecordRejectsInvalidEntry(t *testing.T) {
	repo := &repositoryStub{}
	_, err := New(repo).Record(context.Background(), Caller{ID: "admin", Role: core.RoleSuperAdmin}, domain.Entry{Action: "only action"})
	if !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid entry error = %v", err)
	}
	if repo.recorded.Action != "" {
		t.Fatalf("repository called for invalid entry: %+v", repo.recorded)
	}
}

func TestListAuthorizationFiltersAndPagination(t *testing.T) {
	repo := &repositoryStub{entries: []domain.Entry{{ID: "one"}}, more: true}
	svc := New(repo)
	if _, _, err := svc.List(context.Background(), Caller{Role: core.RoleService}, "", "", 10, 0); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("non-admin list error = %v", err)
	}
	if _, _, err := svc.List(context.Background(), Caller{Role: core.RoleSuperAdmin}, " team ", "", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("partial resource filter error = %v", err)
	}
	for _, tc := range []struct{ limit, offset int }{{0, 0}, {101, 0}, {10, -1}} {
		if _, _, err := svc.List(context.Background(), Caller{Role: core.RoleSuperAdmin}, "", "", tc.limit, tc.offset); !errors.Is(err, core.ErrInvalid) {
			t.Errorf("invalid page %+v error = %v", tc, err)
		}
	}
	items, more, err := svc.List(context.Background(), Caller{Role: core.RoleSuperAdmin}, " team ", " t-1 ", 10, 5)
	if err != nil || !more || len(items) != 1 || repo.listArgs[0] != "team" || repo.listArgs[1] != "t-1" || repo.listArgs[2] != 10 || repo.listArgs[3] != 5 {
		t.Fatalf("list items=%v more=%v args=%v err=%v", items, more, repo.listArgs, err)
	}
	repo.listErr = errors.New("query failed")
	if _, _, err := svc.List(context.Background(), Caller{Role: core.RoleSuperAdmin}, "", "", 10, 0); !errors.Is(err, repo.listErr) {
		t.Fatalf("repository list error = %v", err)
	}
}

func TestRecordAdminActionMapsEvent(t *testing.T) {
	repo := &repositoryStub{}
	when := time.Date(2026, 11, 6, 12, 0, 0, 0, time.FixedZone("test", 3600))
	if err := New(repo).RecordAdminAction(context.Background(), domain.AdminAction{EventID: " 11111111-1111-4111-8111-111111111111 ", ActorID: " admin ", Action: " team.updated ", ResourceType: " team ", ResourceID: " t-1 ", Reason: " corrected ", OccurredAt: when}); err != nil {
		t.Fatalf("record admin action: %v", err)
	}
	got := repo.recorded
	if got.EventID != "11111111-1111-4111-8111-111111111111" || got.ActorID != "admin" || got.Action != "team.updated" || got.ResourceType != "team" || got.ResourceID != "t-1" || got.Reason != "corrected" || !got.OccurredAt.Equal(when.UTC()) {
		t.Fatalf("mapped entry = %+v", got)
	}
	if err := New(&repositoryStub{}).RecordAdminAction(context.Background(), domain.AdminAction{}); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid event error = %v", err)
	}
}

func TestRecordMatchFinalizedMapsActorAndFields(t *testing.T) {
	when := time.Date(2026, 11, 6, 12, 0, 0, 0, time.FixedZone("test", 3600))
	for _, tc := range []struct {
		name, actor string
		want        string
	}{{"official", " official ", "official"}, {"system fallback", "  ", "system"}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &repositoryStub{}
			err := New(repo).RecordMatchFinalized(context.Background(), domain.MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222", FinalizedBy: tc.actor, Reason: " completed ", FinalizedAt: when})
			if err != nil {
				t.Fatalf("record event: %v", err)
			}
			got := repo.recorded
			if got.ActorID != tc.want || got.Action != "match.finalized" || got.ResourceType != "match" || got.ResourceID != "22222222-2222-4222-8222-222222222222" || got.Reason != "completed" || !got.OccurredAt.Equal(when.UTC()) {
				t.Fatalf("mapped entry = %+v", got)
			}
		})
	}
	if err := New(&repositoryStub{}).RecordMatchFinalized(context.Background(), domain.MatchFinalized{}); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("invalid event error = %v", err)
	}
	repo := &repositoryStub{recordErr: errors.New("write failed")}
	if err := New(repo).RecordMatchFinalized(context.Background(), domain.MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", MatchID: "22222222-2222-4222-8222-222222222222", FinalizedAt: when}); !errors.Is(err, repo.recordErr) {
		t.Fatalf("repository error = %v", err)
	}
}
