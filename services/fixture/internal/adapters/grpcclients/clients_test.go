package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
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

type teamServer struct {
	v1.UnimplementedTeamServiceServer
}

func (teamServer) ListTeams(_ context.Context, req *v1.ListTeamsRequest) (*v1.ListTeamsResponse, error) {
	switch {
	case req.GetTournamentId() == "down":
		return nil, status.Error(codes.Unavailable, "no")
	case req.GetPageToken() == "":
		return &v1.ListTeamsResponse{Teams: []*v1.Team{{Id: "a"}, {Id: "b"}}, NextPageToken: "2"}, nil
	}
	return &v1.ListTeamsResponse{Teams: []*v1.Team{{Id: "c"}}}, nil
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
	v1.RegisterTeamServiceServer(server, teamServer{})
	venuev1.RegisterVenueServiceServer(server, venueServer{})
	v1.RegisterMatchServiceServer(server, matchServer{})
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
	if ids, err := NewTeams(conn).IDsInTournament(ctx, "t1"); err != nil || len(ids) != 3 {
		t.Fatalf("teams across pages: %v %v", ids, err)
	}
	if _, err := NewTeams(conn).IDsInTournament(ctx, "down"); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("team outage: %v", err)
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
