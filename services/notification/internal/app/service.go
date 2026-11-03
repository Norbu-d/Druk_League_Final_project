package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
	"github.com/drukleague/platform/services/notification/internal/ports"
)

const ChannelInApp = "in_app"

type Caller struct{ ID, Role string }

type Service struct {
	repo  ports.Repository
	teams ports.TeamDirectory
}

func New(repo ports.Repository, teams ports.TeamDirectory) *Service {
	return &Service{repo: repo, teams: teams}
}

// Send creates an in-app notification for an administrator or internal service.
func (s *Service) Send(ctx context.Context, caller Caller, userID, channel, subject, body string) (domain.Notification, error) {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService {
		return domain.Notification{}, core.Forbidden("only an administrator or service can send notifications")
	}

	item := domain.Notification{
		UserID:  strings.TrimSpace(userID),
		Channel: strings.TrimSpace(channel),
		Subject: strings.TrimSpace(subject),
		Body:    strings.TrimSpace(body),
	}
	if !core.IsUUID(item.UserID) {
		return domain.Notification{}, core.Invalid("user id must be a UUID")
	}
	if item.Channel != ChannelInApp {
		return domain.Notification{}, core.Invalid("only the in_app channel is supported")
	}
	if item.Subject == "" || len(item.Subject) > 200 || item.Body == "" || len(item.Body) > 5000 {
		return domain.Notification{}, core.Invalid("subject and body are required; maximum lengths are 200 and 5000")
	}
	return s.repo.Create(ctx, item)
}

// NotifyMatchFinalized stores one notice per distinct affected team manager.
func (s *Service) NotifyMatchFinalized(ctx context.Context, event domain.FinalizedMatch) error {
	event.EventID = strings.TrimSpace(event.EventID)
	event.MatchID = strings.TrimSpace(event.MatchID)
	event.HomeTeamID = strings.TrimSpace(event.HomeTeamID)
	event.AwayTeamID = strings.TrimSpace(event.AwayTeamID)
	if !core.IsUUID(event.EventID) || !core.IsUUID(event.MatchID) || !core.IsUUID(event.HomeTeamID) || !core.IsUUID(event.AwayTeamID) || event.HomeTeamID == event.AwayTeamID {
		return core.Invalid("event, match, and distinct team ids must be UUIDs")
	}
	if event.HomeScore < 0 || event.HomeScore > 99 || event.AwayScore < 0 || event.AwayScore > 99 {
		return core.Invalid("final scores must be between 0 and 99")
	}
	home, err := s.teams.GetTeam(ctx, event.HomeTeamID)
	if err != nil {
		return err
	}
	away, err := s.teams.GetTeam(ctx, event.AwayTeamID)
	if err != nil {
		return err
	}
	if !core.IsUUID(home.ManagerID) || !core.IsUUID(away.ManagerID) {
		return fmt.Errorf("team manager assignment missing for match %s", event.MatchID)
	}
	users := []string{home.ManagerID}
	if away.ManagerID != home.ManagerID {
		users = append(users, away.ManagerID)
	}
	for _, userID := range users {
		item := domain.Notification{UserID: userID, Channel: ChannelInApp, Subject: "Match finalized", Body: fmt.Sprintf("%s (%s) vs %s (%s) finished %d-%d.", home.Name, home.Dzongkhag, away.Name, away.Dzongkhag, event.HomeScore, event.AwayScore), EventID: event.EventID}
		if _, err := s.repo.Create(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// List returns a user's notices; an administrator may list one user or all users.
func (s *Service) List(ctx context.Context, caller Caller, userID string, size, offset int) ([]domain.Notification, string, error) {
	userID = strings.TrimSpace(userID)
	if !core.IsUUID(caller.ID) {
		return nil, "", core.Forbidden("authenticated user id must be a UUID")
	}
	if caller.Role != core.RoleSuperAdmin {
		if userID != "" && userID != caller.ID {
			return nil, "", core.Forbidden("users can only list their own notifications")
		}
		userID = caller.ID
	}
	if userID != "" && !core.IsUUID(userID) {
		return nil, "", core.Invalid("user id must be a UUID")
	}

	items, more, err := s.repo.List(ctx, userID, size, offset)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if more {
		next = fmt.Sprintf("%d", offset+len(items))
	}
	return items, next, nil
}
