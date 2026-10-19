package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/venue/internal/app"
	"github.com/drukleague/platform/services/venue/internal/domain"
)

type Server struct {
	v1.UnimplementedVenueServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreateVenue(ctx context.Context, req *v1.CreateVenueRequest) (*v1.CreateVenueResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	v, err := s.service.Create(ctx, identity.Role, domain.Venue{Name: req.GetName(), Dzongkhag: req.GetDzongkhag(), Capacity: req.GetCapacity()})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.CreateVenueResponse{Venue: toProto(v)}, nil
}

func (s *Server) GetVenue(ctx context.Context, req *v1.GetVenueRequest) (*v1.GetVenueResponse, error) {
	v, err := s.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetVenueResponse{Venue: toProto(v)}, nil
}

func (s *Server) ListVenues(ctx context.Context, req *v1.ListVenuesRequest) (*v1.ListVenuesResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, more, err := s.service.List(ctx, req.GetDzongkhag(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListVenuesResponse{Venues: make([]*v1.Venue, 0, len(items)), NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, v := range items {
		response.Venues = append(response.Venues, toProto(v))
	}
	return response, nil
}

func toProto(v domain.Venue) *v1.Venue {
	return &v1.Venue{Id: v.ID, Name: v.Name, Dzongkhag: v.Dzongkhag, Capacity: v.Capacity}
}
