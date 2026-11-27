package grpcadapter

import (
	"context"
	"testing"
	"time"

	auditlogv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/audit_log/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/audit-log/internal/app"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type repositoryStub struct {
	recorded         domain.Entry
	entries          []domain.Entry
	more             bool
	listType, listID string
	limit, offset    int
}

func (r *repositoryStub) Record(_ context.Context, item domain.Entry) (domain.Entry, error) {
	r.recorded = item
	item.ID = "entry-1"
	return item, nil
}
func (r *repositoryStub) List(_ context.Context, resourceType, resourceID string, limit, offset int) ([]domain.Entry, bool, error) {
	r.listType, r.listID, r.limit, r.offset = resourceType, resourceID, limit, offset
	return r.entries, r.more, nil
}

func TestRecordAuditEntryUsesAuthenticatedIdentity(t *testing.T) {
	repo := &repositoryStub{}
	server := New(app.New(repo))
	ctx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "admin-1", Role: core.RoleSuperAdmin})
	response, err := server.RecordAuditEntry(ctx, &auditlogv1.RecordAuditEntryRequest{Action: " team.updated ", ResourceType: " team ", ResourceId: " team-1 ", Reason: " fixed "})
	if err != nil {
		t.Fatalf("record RPC: %v", err)
	}
	if repo.recorded.ActorID != "admin-1" || repo.recorded.Action != "team.updated" || repo.recorded.ResourceType != "team" || repo.recorded.ResourceID != "team-1" || repo.recorded.Reason != "fixed" {
		t.Fatalf("stored entry = %+v", repo.recorded)
	}
	if response.GetEntry().GetId() != "entry-1" || response.GetEntry().GetActorId() != "admin-1" || response.GetEntry().GetOccurredAt() == nil {
		t.Fatalf("response = %+v", response)
	}
	if _, err := server.RecordAuditEntry(context.Background(), &auditlogv1.RecordAuditEntryRequest{Action: "x"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unauthenticated caller code = %v, err=%v", status.Code(err), err)
	}
}

func TestListAuditEntriesConvertsPaginationAndEntries(t *testing.T) {
	when := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	repo := &repositoryStub{entries: []domain.Entry{{ID: "e1", ActorID: "admin", Action: "team.updated", ResourceType: "team", ResourceID: "t1", Reason: "reason", OccurredAt: when}}, more: true}
	server := New(app.New(repo))
	ctx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "admin-1", Role: core.RoleSuperAdmin})
	response, err := server.ListAuditEntries(ctx, &auditlogv1.ListAuditEntriesRequest{ResourceType: " team ", ResourceId: " t1 ", PageSize: 2, PageToken: "4"})
	if err != nil {
		t.Fatalf("list RPC: %v", err)
	}
	if repo.listType != "team" || repo.listID != "t1" || repo.limit != 2 || repo.offset != 4 {
		t.Fatalf("repository args: %+v", repo)
	}
	if len(response.GetEntries()) != 1 || response.GetNextPageToken() != "5" {
		t.Fatalf("response = %+v", response)
	}
	item := response.GetEntries()[0]
	if item.GetId() != "e1" || item.GetActorId() != "admin" || item.GetAction() != "team.updated" || item.GetResourceId() != "t1" || item.GetReason() != "reason" || !item.GetOccurredAt().AsTime().Equal(when) {
		t.Fatalf("converted entry = %+v", item)
	}
}

func TestListAuditEntriesRejectsInvalidAccessAndToken(t *testing.T) {
	server := New(app.New(&repositoryStub{}))
	request := &auditlogv1.ListAuditEntriesRequest{}
	ctx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "manager", Role: core.RoleTeamManager})
	if _, err := server.ListAuditEntries(ctx, request); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("manager list code=%v err=%v", status.Code(err), err)
	}
	admin := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "admin", Role: core.RoleSuperAdmin})
	if _, err := server.ListAuditEntries(admin, &auditlogv1.ListAuditEntriesRequest{PageToken: "not-a-number"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid token code=%v err=%v", status.Code(err), err)
	}
	if _, err := server.ListAuditEntries(admin, &auditlogv1.ListAuditEntriesRequest{ResourceType: "team"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("partial filter code=%v err=%v", status.Code(err), err)
	}
}
