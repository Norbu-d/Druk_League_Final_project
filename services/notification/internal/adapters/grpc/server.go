package grpcadapter

import (
	"context"

	notificationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/notification/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/notification/internal/app"
	"github.com/drukleague/platform/services/notification/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	notificationv1.UnimplementedNotificationServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	item, err := s.service.Send(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, req.GetUserId(), req.GetChannel(), req.GetSubject(), req.GetBody())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &notificationv1.SendNotificationResponse{Notification: toProto(item)}, nil
}

func (s *Server) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	limit, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	identity, _ := servicekit.IdentityFromContext(ctx)
	items, more, err := s.service.List(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, req.GetUserId(), int(limit), offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &notificationv1.ListNotificationsResponse{
		Notifications: make([]*notificationv1.Notification, 0, len(items)),
		NextPageToken: servicekit.NextPageToken(offset, len(items), more),
	}
	for _, item := range items {
		response.Notifications = append(response.Notifications, toProto(item))
	}
	return response, nil
}

func toProto(item domain.Notification) *notificationv1.Notification {
	return &notificationv1.Notification{Id: item.ID, UserId: item.UserID, Channel: item.Channel, Subject: item.Subject, Body: item.Body, CreatedAt: timestamppb.New(item.CreatedAt)}
}
