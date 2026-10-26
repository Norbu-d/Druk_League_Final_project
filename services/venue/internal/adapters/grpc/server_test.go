package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/venue/internal/app"
	"github.com/drukleague/platform/services/venue/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const venueID = "11111111-1111-4111-8111-111111111111"

type repo struct{}

func (repo) Create(_ context.Context, v domain.Venue) (domain.Venue, error) {
	v.ID = venueID
	return v, nil
}
func (repo) Get(_ context.Context, id string) (domain.Venue, error) {
	return domain.Venue{ID: id, Name: "Ground"}, nil
}
func (repo) List(context.Context, string, int, int) ([]domain.Venue, bool, error) {
	return []domain.Venue{{ID: venueID}, {ID: venueID}}, true, nil
}

type dz struct{}

func (dz) Canonical(context.Context, string) (string, error) { return "Thimphu", nil }

func admin() context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "a", Role: core.RoleSuperAdmin})
}

func TestVenueRPCs(t *testing.T) {
	s := New(app.New(repo{}, dz{}))
	created, err := s.CreateVenue(admin(), &v1.CreateVenueRequest{Name: "Ground", Dzongkhag: "thimphu", Capacity: 100})
	if err != nil || created.GetVenue().GetDzongkhag() != "Thimphu" {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateVenue(context.Background(), &v1.CreateVenueRequest{Name: "Ground", Dzongkhag: "Thimphu"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous create: %v", err)
	}
	if got, err := s.GetVenue(admin(), &v1.GetVenueRequest{Id: venueID}); err != nil || got.GetVenue().GetName() != "Ground" {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.GetVenue(admin(), &v1.GetVenueRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	list, err := s.ListVenues(admin(), &v1.ListVenuesRequest{PageSize: 2})
	if err != nil || len(list.GetVenues()) != 2 || list.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := s.ListVenues(admin(), &v1.ListVenuesRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad token accepted")
	}
}
