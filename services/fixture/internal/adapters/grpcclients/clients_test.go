package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	registrationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type tournamentServer struct {
	v1.UnimplementedTournamentServiceServer
}

func (tournamentServer) GetTournament(_ context.Context, req *v1.GetTournamentRequest) (*v1.GetTournamentResponse, error) {
	if req.GetId() == "missing" {
		return nil, status.Error(codes.NotFound, "no")
	}
	return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: req.GetId(), Dzongkhag: "Paro"}}, nil
}

type registrationServer struct {
	registrationv1.UnimplementedRegistrationServiceServer
}

func (registrationServer) ListRegistrations(_ context.Context, req *registrationv1.ListRegistrationsRequest) (*registrationv1.ListRegistrationsResponse, error) {
	switch {
	case req.GetTournamentId() == "down":
		return nil, status.Error(codes.Unavailable, "no")
	case req.GetStatus() != "approved":
		return nil, status.Error(codes.InvalidArgument, "status must be approved")
	case req.GetPageToken() == "":
		return &registrationv1.ListRegistrationsResponse{Registrations: []*registrationv1.Registration{{TeamId: "a", Status: "approved"}, {TeamId: "b", Status: "approved"}}, NextPageToken: "2"}, nil
	}
	return &registrationv1.ListRegistrationsResponse{Registrations: []*registrationv1.Registration{{TeamId: "c", Status: "approved"}}}, nil
}

type venueServer struct {
	venuev1.UnimplementedVenueServiceServer
}

func (venueServer) ListVenues(_ context.Context, req *venuev1.ListVenuesRequest) (*venuev1.ListVenuesResponse, error) {
	switch req.GetDzongkhag() {
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	case "bad":
		return nil, status.Error(codes.InvalidArgument, "no")
	}
	return &venuev1.ListVenuesResponse{Venues: []*venuev1.Venue{{Id: "v2", Name: "Paro Ground"}, {Id: "v1", Name: "Bondey Ground"}}}, nil
}

type matchServer struct {
	v1.UnimplementedMatchServiceServer
}

type existingMatchServer struct {
	v1.UnimplementedMatchServiceServer
}

func (existingMatchServer) CreateMatch(context.Context, *v1.CreateMatchRequest) (*v1.CreateMatchResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "duplicate request")
}

func (existingMatchServer) ListMatches(_ context.Context, _ *v1.ListMatchesRequest) (*v1.ListMatchesResponse, error) {
	return &v1.ListMatchesResponse{Matches: []*v1.Match{{Id: "existing-match", TournamentId: "11111111-1111-4111-8111-111111111111", HomeTeamId: "22222222-2222-4222-8222-222222222222", AwayTeamId: "33333333-3333-4333-8333-333333333333", Venue: "Paro Ground", Kickoff: timestamppb.New(time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC))}}}, nil
}

type standingsServer struct {
	standingsv1.UnimplementedStandingsServiceServer
}

func (standingsServer) GetStandings(_ context.Context, req *standingsv1.GetStandingsRequest) (*standingsv1.GetStandingsResponse, error) {
	return &standingsv1.GetStandingsResponse{Standings: []*standingsv1.Standing{{TeamId: req.GetTournamentId()}, {TeamId: "team-two"}}}, nil
}

func (matchServer) CreateMatch(_ context.Context, req *v1.CreateMatchRequest) (*v1.CreateMatchResponse, error) {
	switch req.GetHomeTeamId() {
	case "clash":
		return nil, status.Error(codes.AlreadyExists, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	}
	return &v1.CreateMatchResponse{Match: &v1.Match{Id: "m1"}}, nil
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterTournamentServiceServer(server, tournamentServer{})
	registrationv1.RegisterRegistrationServiceServer(server, registrationServer{})
	venuev1.RegisterVenueServiceServer(server, venueServer{})
	v1.RegisterMatchServiceServer(server, matchServer{})
	standingsv1.RegisterStandingsServiceServer(server, standingsServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestClients(t *testing.T) {
	conn := dial(t)
	ctx := context.Background()
	if tournament, err := NewTournaments(conn).Get(ctx, "t1"); err != nil || tournament.Dzongkhag != "Paro" {
		t.Fatalf("tournament: %+v %v", tournament, err)
	}
	if _, err := NewTournaments(conn).Get(ctx, "missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing tournament: %v", err)
	}
	if ids, err := NewRegistrations(conn).ApprovedTeamIDs(ctx, "t1"); err != nil || len(ids) != 3 || ids[0] != "a" || ids[2] != "c" {
		t.Fatalf("approved registrations across pages: %v %v", ids, err)
	}
	if _, err := NewRegistrations(conn).ApprovedTeamIDs(ctx, "down"); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("registration service outage: %v", err)
	}
	if ids, err := NewStandings(conn).TeamIDsByRank(ctx, "ranked"); err != nil || len(ids) != 2 || ids[0] != "ranked" {
		t.Fatalf("standings order: %v %v", ids, err)
	}
	venues, err := NewVenues(conn).InDzongkhag(ctx, "Paro")
	if err != nil || len(venues) != 2 || venues[0].Name != "Bondey Ground" {
		t.Fatalf("venues sorted by name: %+v %v", venues, err)
	}
	if venues, err := NewVenues(conn).InDzongkhag(ctx, "down"); err != nil || venues != nil {
		t.Fatalf("venue outage must degrade to no venues: %v %v", venues, err)
	}
	if _, err := NewVenues(conn).InDzongkhag(ctx, "bad"); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("bad venue filter: %v", err)
	}
	fixture := domain.Fixture{TournamentID: "t1", HomeTeamID: "a", AwayTeamID: "b", Kickoff: time.Now()}
	if id, err := NewMatches(conn).Create(ctx, fixture); err != nil || id != "m1" {
		t.Fatalf("create match: %q %v", id, err)
	}
	fixture.HomeTeamID = "clash"
	if _, err := NewMatches(conn).Create(ctx, fixture); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("clash: %v", err)
	}
	fixture.HomeTeamID = "down"
	if _, err := NewMatches(conn).Create(ctx, fixture); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("match outage: %v", err)
	}
}

func TestMatchCreateRecoversExistingFixtureAfterRetry(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterMatchServiceServer(server, existingMatchServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	fixture := domain.Fixture{TournamentID: "11111111-1111-4111-8111-111111111111", HomeTeamID: "22222222-2222-4222-8222-222222222222", AwayTeamID: "33333333-3333-4333-8333-333333333333", VenueName: "Paro Ground", Kickoff: time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)}
	if id, err := NewMatches(conn).Create(context.Background(), fixture); err != nil || id != "existing-match" {
		t.Fatalf("idempotent match create: %q %v", id, err)
	}
}
