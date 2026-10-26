package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/player/internal/domain"
	"github.com/drukleague/platform/services/player/internal/ports"
)

// Caller is the authenticated user making the request.
type Caller struct{ ID, Role string }

type Service struct {
	repo  ports.Repository
	teams ports.Teams
}

func New(repo ports.Repository, teams ports.Teams) *Service {
	return &Service{repo: repo, teams: teams}
}

// Create adds a player to a team. Only the admin or the team's own manager may change a roster.
func (s *Service) Create(ctx context.Context, caller Caller, p domain.Player) (domain.Player, error) {
	if err := canEditRosters(caller); err != nil {
		return domain.Player{}, err
	}
	p, err := p.Normalize()
	if err != nil {
		return domain.Player{}, err
	}
	if err := s.authorizeTeam(ctx, caller, p.TeamID); err != nil {
		return domain.Player{}, err
	}
	return s.repo.Create(ctx, p)
}

// Update changes a player's name and shirt number; the player stays on the same team.
func (s *Service) Update(ctx context.Context, caller Caller, p domain.Player) (domain.Player, error) {
	if err := canEditRosters(caller); err != nil {
		return domain.Player{}, err
	}
	existing, err := s.Get(ctx, p.ID)
	if err != nil {
		return domain.Player{}, err
	}
	p.ID, p.TeamID = existing.ID, existing.TeamID
	if p, err = p.Normalize(); err != nil {
		return domain.Player{}, err
	}
	if err := s.authorizeTeam(ctx, caller, p.TeamID); err != nil {
		return domain.Player{}, err
	}
	return s.repo.Update(ctx, p)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Player, error) {
	if !core.IsUUID(id) {
		return domain.Player{}, core.Invalid("player id must be a UUID")
	}
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, teamID string, limit, offset int) ([]domain.Player, bool, error) {
	teamID = strings.TrimSpace(teamID)
	if teamID != "" && !core.IsUUID(teamID) {
		return nil, false, core.Invalid("team id must be a UUID")
	}
	return s.repo.List(ctx, teamID, limit, offset)
}

func canEditRosters(caller Caller) error {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleTeamManager {
		return core.Forbidden("only the tournament admin or the team's manager can change players")
	}
	return nil
}

// authorizeTeam confirms the team exists and, for a team manager, that they manage it.
func (s *Service) authorizeTeam(ctx context.Context, caller Caller, teamID string) error {
	team, err := s.teams.Get(ctx, teamID)
	if err != nil {
		return err
	}
	if caller.Role == core.RoleTeamManager && (team.ManagerID == "" || team.ManagerID != caller.ID) {
		return core.Forbidden("you do not manage this team")
	}
	return nil
}
