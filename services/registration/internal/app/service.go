package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"github.com/drukleague/platform/services/registration/internal/ports"
)

// maxReasonLength keeps review reasons short enough for notifications and audit entries.
const maxReasonLength = 500

type Caller struct{ ID, Role string }

type Service struct {
	repo        ports.Repository
	teams       ports.Teams
	tournaments ports.Tournaments
}

func New(repo ports.Repository, teams ports.Teams, tournaments ports.Tournaments) *Service {
	return &Service{repo: repo, teams: teams, tournaments: tournaments}
}

// Register enters a team into a tournament. The tournament admin may register any team;
// a team manager may register only a team they manage. Both the team and the tournament
// must exist, and the tournament must still be open for registration.
func (s *Service) Register(ctx context.Context, caller Caller, tournamentID, teamID string) (domain.Registration, error) {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleTeamManager {
		return domain.Registration{}, core.Forbidden("only the tournament admin or a team manager can register a team")
	}
	tournamentID, teamID = strings.TrimSpace(tournamentID), strings.TrimSpace(teamID)
	if !core.IsUUID(tournamentID) || !core.IsUUID(teamID) {
		return domain.Registration{}, core.Invalid("tournament id and team id must be UUIDs")
	}
	team, err := s.teams.Get(ctx, teamID)
	if err != nil {
		return domain.Registration{}, err
	}
	tournament, err := s.tournaments.Get(ctx, tournamentID)
	if err != nil {
		return domain.Registration{}, err
	}
	if tournament.Status != domain.TournamentOpen {
		return domain.Registration{}, core.Precondition("registration is not open for this tournament")
	}
	if caller.Role == core.RoleTeamManager && (team.ManagerID == "" || team.ManagerID != caller.ID) {
		return domain.Registration{}, core.Forbidden("only the team's manager can register it")
	}
	return s.repo.Create(ctx, domain.Registration{TournamentID: tournamentID, TeamID: teamID, RegisteredBy: caller.ID})
}

// Review approves or rejects a pending registration. Only the tournament admin may review,
// and a rejection must say why.
func (s *Service) Review(ctx context.Context, caller Caller, registrationID string, approved bool, reason string) (domain.Registration, error) {
	if caller.Role != core.RoleSuperAdmin {
		return domain.Registration{}, core.Forbidden("only the tournament admin can review registrations")
	}
	registrationID, reason = strings.TrimSpace(registrationID), strings.TrimSpace(reason)
	if !core.IsUUID(registrationID) {
		return domain.Registration{}, core.Invalid("registration id must be a UUID")
	}
	if !approved && reason == "" {
		return domain.Registration{}, core.Invalid("a reason is required when rejecting a registration")
	}
	if len(reason) > maxReasonLength {
		return domain.Registration{}, core.Invalid("reason is too long")
	}
	decision := domain.StatusRejected
	if approved {
		decision = domain.StatusApproved
	}
	return s.repo.Review(ctx, registrationID, decision, reason, caller.ID)
}

// List returns registrations. The admin and internal services see all of them; a team
// manager sees only the ones they submitted.
func (s *Service) List(ctx context.Context, caller Caller, tournamentID, status string, limit, offset int) ([]domain.Registration, bool, error) {
	filter := ports.Filter{TournamentID: strings.TrimSpace(tournamentID), Status: strings.TrimSpace(status)}
	switch caller.Role {
	case core.RoleSuperAdmin, core.RoleService:
	case core.RoleTeamManager:
		filter.RegisteredBy = caller.ID
	default:
		return nil, false, core.Forbidden("not allowed to list registrations")
	}
	if filter.TournamentID != "" && !core.IsUUID(filter.TournamentID) {
		return nil, false, core.Invalid("tournament id must be a UUID")
	}
	if filter.Status != "" && !domain.ValidStatus(filter.Status) {
		return nil, false, core.Invalid("status must be pending, approved, or rejected")
	}
	return s.repo.List(ctx, filter, limit, offset)
}
