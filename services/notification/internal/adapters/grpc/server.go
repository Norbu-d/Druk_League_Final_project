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

func New(service *app.Service) *Server {
	return &Server{service: service}
}

func caller(ctx context.Context) app.Caller {
	identity, _ := servicekit.IdentityFromContext(ctx)
	return app.Caller{ID: identity.ID, Role: identity.Role}
}

func (s *Server) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	item, err := s.service.Send(ctx, caller(ctx), req.GetUserId(), req.GetChannel(), req.GetSubject(), req.GetBody())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &notificationv1.SendNotificationResponse{Notification: toProto(item)}, nil
}

func (s *Server) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, next, err := s.service.List(ctx, caller(ctx), req.GetUserId(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &notificationv1.ListNotificationsResponse{
		Notifications: make([]*notificationv1.Notification, 0, len(items)),
		NextPageToken: next,
	}
	for _, item := range items {
		response.Notifications = append(response.Notifications, toProto(item))
	}
	return response, nil
}

func toProto(item domain.Notification) *notificationv1.Notification {
	return &notificationv1.Notification{
		Id: item.ID, UserId: item.UserID, Channel: item.Channel, Subject: item.Subject,
		Body: item.Body, CreatedAt: timestamppb.New(item.CreatedAt),
	}
}
