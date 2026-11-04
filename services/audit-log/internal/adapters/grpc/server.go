package grpcadapter

import (
	"context"

	auditlogv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/audit_log/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/audit-log/internal/app"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	auditlogv1.UnimplementedAuditLogServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) RecordAuditEntry(ctx context.Context, req *auditlogv1.RecordAuditEntryRequest) (*auditlogv1.RecordAuditEntryResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	item, err := s.service.Record(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, domain.Entry{
		Action: req.GetAction(), ResourceType: req.GetResourceType(), ResourceID: req.GetResourceId(), Reason: req.GetReason(),
	})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &auditlogv1.RecordAuditEntryResponse{Entry: toProto(item)}, nil
}

func (s *Server) ListAuditEntries(ctx context.Context, req *auditlogv1.ListAuditEntriesRequest) (*auditlogv1.ListAuditEntriesResponse, error) {
	limit, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	identity, _ := servicekit.IdentityFromContext(ctx)
	items, more, err := s.service.List(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, req.GetResourceType(), req.GetResourceId(), limit, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &auditlogv1.ListAuditEntriesResponse{Entries: make([]*auditlogv1.AuditEntry, 0, len(items)),
		NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, item := range items {
		response.Entries = append(response.Entries, toProto(item))
	}
	return response, nil
}

func toProto(item domain.Entry) *auditlogv1.AuditEntry {
	return &auditlogv1.AuditEntry{Id: item.ID, ActorId: item.ActorID, Action: item.Action, ResourceType: item.ResourceType,
		ResourceId: item.ResourceID, Reason: item.Reason, OccurredAt: timestamppb.New(item.OccurredAt)}
}
