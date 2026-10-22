package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/team/internal/domain"
	"github.com/drukleague/platform/services/team/internal/ports"
)

// Caller is the authenticated user making the request.
type Caller struct{ ID, Role string }

type Service struct {
	repo        ports.Repository
	tournaments ports.Tournaments
	dzongkhags  ports.Dzongkhags
}

func New(repo ports.Repository, tournaments ports.Tournaments, dzongkhags ports.Dzongkhags) *Service {
	return &Service{repo: repo, tournaments: tournaments, dzongkhags: dzongkhags}
}

// Create adds a team. The admin can create any team; a team manager becomes the manager of
// the team they create and may only enter it into a tournament that is open for registration.
func (s *Service) Create(ctx context.Context, caller Caller, t domain.Team) (domain.Team, error) {
	switch caller.Role {
	case core.RoleSuperAdmin:
		t.ManagerID = ""
	case core.RoleTeamManager:
		t.ManagerID = caller.ID
	default:
		return domain.Team{}, core.Forbidden("only the tournament admin or a team manager can create teams")
	}
	t, err := t.Normalize()
	if err != nil {
		return domain.Team{}, err
	}
	if t.Dzongkhag, err = s.dzongkhags.Canonical(ctx, t.Dzongkhag); err != nil {
		return domain.Team{}, err
	}
	if t.TournamentID != "" {
		tournament, err := s.tournaments.Get(ctx, t.TournamentID)
		if err != nil {
			return domain.Team{}, err
		}
		if caller.Role == core.RoleTeamManager && tournament.Status != domain.TournamentOpen {
			return domain.Team{}, core.Precondition("tournament is not open for registration")
		}
	}
	return s.repo.Create(ctx, t)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Team, error) {
	if !core.IsUUID(id) {
		return domain.Team{}, core.Invalid("team id must be a UUID")
	}
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, filter domain.Filter, limit, offset int) ([]domain.Team, bool, error) {
	filter = domain.Filter{TournamentID: strings.TrimSpace(filter.TournamentID), Dzongkhag: strings.TrimSpace(filter.Dzongkhag), ManagerID: strings.TrimSpace(filter.ManagerID)}
	if filter.TournamentID != "" && !core.IsUUID(filter.TournamentID) {
		return nil, false, core.Invalid("tournament id must be a UUID")
	}
	if filter.ManagerID != "" && !core.IsUUID(filter.ManagerID) {
		return nil, false, core.Invalid("manager id must be a UUID")
	}
	if filter.Dzongkhag != "" {
		canonical, err := s.dzongkhags.Canonical(ctx, filter.Dzongkhag)
		if err != nil {
			return nil, false, err
		}
		filter.Dzongkhag = canonical
	}
	return s.repo.List(ctx, filter, limit, offset)
}
