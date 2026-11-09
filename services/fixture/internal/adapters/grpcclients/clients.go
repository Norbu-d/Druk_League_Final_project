package grpcclients

import (
	"context"
	"sort"

	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// maxPages stops a paging loop if a dependency keeps returning a next-page token.
const maxPages = 50

type Tournaments struct{ client v1.TournamentServiceClient }

func NewTournaments(conn grpc.ClientConnInterface) *Tournaments {
	return &Tournaments{client: v1.NewTournamentServiceClient(conn)}
}

func (t *Tournaments) Get(ctx context.Context, id string) (domain.Tournament, error) {
	response, err := t.client.GetTournament(ctx, &v1.GetTournamentRequest{Id: id})
	if err != nil {
		return domain.Tournament{}, servicekit.FromStatus(err, "tournament")
	}
	return domain.Tournament{ID: response.GetTournament().GetId(), Dzongkhag: response.GetTournament().GetDzongkhag()}, nil
}

type Registrations struct {
	client registrationv1.RegistrationServiceClient
}

func NewRegistrations(conn grpc.ClientConnInterface) *Registrations {
	return &Registrations{client: registrationv1.NewRegistrationServiceClient(conn)}
}

func (r *Registrations) ApprovedTeamIDs(ctx context.Context, tournamentID string) ([]string, error) {
	ids := []string{}
	token := ""
	for page := 0; page < maxPages; page++ {
		response, err := r.client.ListRegistrations(ctx, &registrationv1.ListRegistrationsRequest{TournamentId: tournamentID, Status: "approved", PageSize: 100, PageToken: token})
		if err != nil {
			return nil, servicekit.FromStatus(err, "approved registrations")
		}
		for _, registration := range response.GetRegistrations() {
			ids = append(ids, registration.GetTeamId())
		}
		if token = response.GetNextPageToken(); token == "" {
			return ids, nil
		}
	}
	return nil, core.Dependency("approved registration listing exceeded pagination limit")
}

type Venues struct{ client venuev1.VenueServiceClient }

func NewVenues(conn grpc.ClientConnInterface) *Venues {
	return &Venues{client: venuev1.NewVenueServiceClient(conn)}
}

// InDzongkhag returns the tournament district's venues. Venues are optional for a schedule,
// so a Venue service outage degrades to fixtures without a venue instead of failing.
func (v *Venues) InDzongkhag(ctx context.Context, dzongkhag string) ([]domain.Venue, error) {
	response, err := v.client.ListVenues(ctx, &venuev1.ListVenuesRequest{Dzongkhag: dzongkhag, PageSize: 100})
	if err != nil {
		if code := status.Code(err); code == codes.Unavailable || code == codes.DeadlineExceeded {
			return nil, nil
		}
		return nil, servicekit.FromStatus(err, "venues")
	}
	venues := make([]domain.Venue, 0, len(response.GetVenues()))
	for _, venue := range response.GetVenues() {
		venues = append(venues, domain.Venue{ID: venue.GetId(), Name: venue.GetName()})
	}
	sort.Slice(venues, func(i, j int) bool { return venues[i].Name < venues[j].Name })
	return venues, nil
}

type Matches struct{ client v1.MatchServiceClient }

func NewMatches(conn grpc.ClientConnInterface) *Matches {
	return &Matches{client: v1.NewMatchServiceClient(conn)}
}

func (m *Matches) Create(ctx context.Context, f domain.Fixture) (string, error) {
	response, err := m.client.CreateMatch(ctx, &v1.CreateMatchRequest{TournamentId: f.TournamentID, HomeTeamId: f.HomeTeamID, AwayTeamId: f.AwayTeamID, Venue: f.VenueName, Kickoff: timestamppb.New(f.Kickoff)})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			if id, found, lookupErr := m.findExact(ctx, f); lookupErr == nil && found {
				return id, nil
			}
			return "", core.Conflict("a team or venue already has a match at " + f.Kickoff.Format("2006-01-02 15:04") + " UTC")
		}
		return "", servicekit.FromStatus(err, "match")
	}
	return response.GetMatch().GetId(), nil
}

func (m *Matches) findExact(ctx context.Context, f domain.Fixture) (string, bool, error) {
	token := ""
	for page := 0; page < maxPages; page++ {
		response, err := m.client.ListMatches(ctx, &v1.ListMatchesRequest{TournamentId: f.TournamentID, PageSize: 100, PageToken: token})
		if err != nil {
			return "", false, servicekit.FromStatus(err, "matches")
		}
		for _, match := range response.GetMatches() {
			if match.GetHomeTeamId() == f.HomeTeamID && match.GetAwayTeamId() == f.AwayTeamID && match.GetVenue() == f.VenueName &&
				match.GetKickoff() != nil && match.GetKickoff().AsTime().Equal(f.Kickoff) {
				return match.GetId(), true, nil
			}
		}
		token = response.GetNextPageToken()
		if token == "" {
			return "", false, nil
		}
	}
	return "", false, core.Dependency("match listing exceeded pagination limit")
}

type Standings struct {
	client standingsv1.StandingsServiceClient
}

func NewStandings(conn grpc.ClientConnInterface) *Standings {
	return &Standings{client: standingsv1.NewStandingsServiceClient(conn)}
}

func (s *Standings) TeamIDsByRank(ctx context.Context, tournamentID string) ([]string, error) {
	response, err := s.client.GetStandings(ctx, &standingsv1.GetStandingsRequest{TournamentId: tournamentID})
	if err != nil {
		return nil, servicekit.FromStatus(err, "standings")
	}
	ids := make([]string, 0, len(response.GetStandings()))
	for _, standing := range response.GetStandings() {
		ids = append(ids, standing.GetTeamId())
	}
	return ids, nil
}
