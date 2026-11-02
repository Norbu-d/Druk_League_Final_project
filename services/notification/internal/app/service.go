package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
	"github.com/drukleague/platform/services/notification/internal/ports"
)

const inAppChannel = "in_app"

type Caller struct{ ID, Role string }

type Service struct {
	repo  ports.Repository
	teams ports.TeamGetter
}

func New(repo ports.Repository, teams ports.TeamGetter) *Service {
	return &Service{repo: repo, teams: teams}
}

func (s *Service) Send(ctx context.Context, caller Caller, userID, channel, subject, body string) (domain.Notification, error) {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService {
		return domain.Notification{}, core.Forbidden("only admins and internal services can send notifications")
	}
	userID = strings.TrimSpace(userID)
	channel = strings.ToLower(strings.TrimSpace(channel))
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if !core.IsUUID(userID) {
		return domain.Notification{}, core.Invalid("user id must be a UUID")
	}
	if channel != inAppChannel {
		return domain.Notification{}, core.Invalid("channel must be in_app")
	}
	if subject == "" || len(subject) > 200 || body == "" || len(body) > 4000 {
		return domain.Notification{}, core.Invalid("subject and body are required and must fit their limits")
	}
	item, _, err := s.repo.Create(ctx, domain.Notification{UserID: userID, Channel: channel, Subject: subject, Body: body, Status: "pending"})
	return item, err
}

func (s *Service) List(ctx context.Context, caller Caller, userID string, limit, offset int) ([]domain.Notification, bool, error) {
	userID = strings.TrimSpace(userID)
	if !core.IsUUID(userID) {
		return nil, false, core.Invalid("user id must be a UUID")
	}
	if caller.Role != core.RoleSuperAdmin && caller.ID != userID {
		return nil, false, core.Forbidden("cannot read another user's notifications")
	}
	return s.repo.List(ctx, userID, limit, offset)
}

// NotifyMatchFinalized creates one idempotent in-app notice for each distinct team manager.
// Dzongkhag names are included in the message; Jira has no district-recipient identities defined.
func (s *Service) NotifyMatchFinalized(ctx context.Context, event domain.MatchFinalized) error {
	if err := event.Validate(); err != nil {
		return err
	}
	home, err := s.teams.Get(ctx, event.HomeTeamID)
	if err != nil {
		return err
	}
	away, err := s.teams.Get(ctx, event.AwayTeamID)
	if err != nil {
		return err
	}
	if !validEventTeam(home, event.TournamentID, event.HomeTeamID) || !validEventTeam(away, event.TournamentID, event.AwayTeamID) {
		return core.Invalid("event teams do not match the finalized tournament")
	}
	if !core.IsUUID(home.ManagerID) || !core.IsUUID(away.ManagerID) {
		return core.Precondition("both participating teams need an assigned manager before notifications can be delivered")
	}
	body := fmt.Sprintf("Final result: %s %d–%d %s. Tournament %s. Dzongkhags: %s and %s. Finalized at %s.",
		home.Name, event.HomeScore, event.AwayScore, away.Name, event.TournamentID, home.Dzongkhag, away.Dzongkhag, event.FinalizedAt.UTC().Format("2006-01-02 15:04 UTC"))
	recipients := []string{home.ManagerID}
	if away.ManagerID != home.ManagerID {
		recipients = append(recipients, away.ManagerID)
	}
	for _, userID := range recipients {
		if _, _, err := s.repo.Create(ctx, domain.Notification{
			EventID: event.EventID, UserID: userID, Channel: inAppChannel,
			Subject: "Match result finalized", Body: body, Status: "pending",
		}); err != nil {
			return err
		}
	}
	return nil
}

func validEventTeam(team domain.Team, tournamentID, teamID string) bool {
	return team.ID == teamID && (team.TournamentID == "" || team.TournamentID == tournamentID) &&
		strings.TrimSpace(team.Name) != "" && strings.TrimSpace(team.Dzongkhag) != ""
}
