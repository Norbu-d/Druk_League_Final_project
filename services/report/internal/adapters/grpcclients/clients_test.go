package grpcclients

import (
	"context"
	"net"
	"testing"
	"time"

	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeReportSources struct {
	standingsv1.UnimplementedStandingsServiceServer
	fixturev1.UnimplementedFixtureServiceServer
}

func (fakeReportSources) GetStandings(context.Context, *standingsv1.GetStandingsRequest) (*standingsv1.GetStandingsResponse, error) {
	return &standingsv1.GetStandingsResponse{Standings: []*standingsv1.Standing{{TeamId: "team-id", TeamName: "Thimphu", Points: 3}}}, nil
}

func (fakeReportSources) ListFixtures(context.Context, *fixturev1.ListFixturesRequest) (*fixturev1.ListFixturesResponse, error) {
	return &fixturev1.ListFixturesResponse{Fixtures: []*fixturev1.Fixture{{Id: "fixture-id", TournamentId: "tournament-id", Kickoff: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)), Round: 1}}, NextPageToken: "next"}, nil
}

func TestStandingsAndFixtureClientsMapResponses(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	sources := fakeReportSources{}
	standingsv1.RegisterStandingsServiceServer(server, sources)
	fixturev1.RegisterFixtureServiceServer(server, sources)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///report-test", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	standings, err := NewStandings(conn).Get(ctx, "tournament-id")
	if err != nil || len(standings) != 1 || standings[0].TeamName != "Thimphu" || standings[0].Points != 3 {
		t.Fatalf("standings mapping: rows=%+v err=%v", standings, err)
	}
	fixtures, next, err := NewFixtures(conn).List(ctx, "tournament-id", 100, "")
	if err != nil || next != "next" || len(fixtures) != 1 || fixtures[0].ID != "fixture-id" || fixtures[0].Kickoff.IsZero() {
		t.Fatalf("fixture mapping: rows=%+v next=%q err=%v", fixtures, next, err)
	}
}
