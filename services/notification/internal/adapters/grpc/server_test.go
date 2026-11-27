package grpcadapter

import (
	"context"
	"testing"
	"time"

	notificationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/notification/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/notification/internal/app"
	"github.com/drukleague/platform/services/notification/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const recipientID = "11111111-1111-4111-8111-111111111111"

type memoryRepo struct{ items []domain.Notification }

func (r *memoryRepo) Create(_ context.Context, item domain.Notification) (domain.Notification, bool, error) {
	item.ID = "22222222-2222-4222-8222-222222222222"
	item.CreatedAt = time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	r.items = append(r.items, item)
	return item, true, nil
}

func (r *memoryRepo) List(_ context.Context, userID string, limit, offset int) ([]domain.Notification, bool, error) {
	var items []domain.Notification
	for _, item := range r.items {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	if offset >= len(items) {
		return []domain.Notification{}, false, nil
	}
	end := min(offset+limit, len(items))
	return items[offset:end], end < len(items), nil
}

type noTeams struct{}

func (noTeams) Get(context.Context, string) (domain.Team, error) { return domain.Team{}, nil }

func TestNotificationRPCs(t *testing.T) {
	repo := &memoryRepo{}
	server := New(app.New(repo, noTeams{}))
	serviceCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "service:tests", Role: core.RoleService})
	resp, err := server.SendNotification(serviceCtx, &notificationv1.SendNotificationRequest{UserId: recipientID, Channel: "in_app", Subject: "Result", Body: "Final result is ready."})
	if err != nil || resp.GetNotification().GetUserId() != recipientID || resp.GetNotification().GetCreatedAt() == nil {
		t.Fatalf("send RPC: response=%+v err=%v", resp, err)
	}
	if _, err := server.SendNotification(servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: recipientID, Role: core.RoleViewer}), &notificationv1.SendNotificationRequest{UserId: recipientID, Channel: "in_app", Subject: "Result", Body: "Body"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unauthorized send code = %v, want permission denied", status.Code(err))
	}
	listCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: recipientID, Role: core.RoleTeamManager})
	listed, err := server.ListNotifications(listCtx, &notificationv1.ListNotificationsRequest{UserId: recipientID, PageSize: 10})
	if err != nil || len(listed.GetNotifications()) != 1 || listed.GetNotifications()[0].GetSubject() != "Result" || listed.GetNextPageToken() != "" {
		t.Fatalf("list own notifications: response=%+v err=%v", listed, err)
	}
	if _, err := server.ListNotifications(listCtx, &notificationv1.ListNotificationsRequest{UserId: "33333333-3333-4333-8333-333333333333"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("other user's list code = %v, want permission denied", status.Code(err))
	}
	if _, err := server.ListNotifications(listCtx, &notificationv1.ListNotificationsRequest{UserId: recipientID, PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad page token code = %v, want invalid argument", status.Code(err))
	}
}
