package grpcadapter

import (
	"context"
	"errors"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/dzongkhag/v1"
	"github.com/drukleague/platform/services/dzongkhag/internal/app"
	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedDzongkhagServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }
func (s *Server) GetDzongkhag(ctx context.Context, req *v1.GetDzongkhagRequest) (*v1.GetDzongkhagResponse, error) {
	d, err := s.service.Get(ctx, req.GetCode())
	if errors.Is(err, domain.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "dzongkhag not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not read dzongkhag")
	}
	return &v1.GetDzongkhagResponse{Dzongkhag: &v1.Dzongkhag{Code: d.Code, Name: d.Name}}, nil
}
func (s *Server) ListDzongkhags(ctx context.Context, _ *v1.ListDzongkhagsRequest) (*v1.ListDzongkhagsResponse, error) {
	items, err := s.service.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list dzongkhags")
	}
	result := &v1.ListDzongkhagsResponse{Dzongkhags: make([]*v1.Dzongkhag, 0, len(items))}
	for _, d := range items {
		result.Dzongkhags = append(result.Dzongkhags, &v1.Dzongkhag{Code: d.Code, Name: d.Name})
	}
	return result, nil
}
