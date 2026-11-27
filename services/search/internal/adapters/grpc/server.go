package grpcadapter

import (
	"context"
	"errors"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/search/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/search/internal/app"
	"github.com/drukleague/platform/services/search/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedSearchServiceServer
	service *app.Service
}

func New(repository appRepository) *Server { return &Server{service: app.New(repository)} }

type appRepository interface {
	Search(context.Context, string, []string, int, int) ([]domain.Hit, error)
}

func (s *Server) Search(ctx context.Context, request *v1.SearchRequest) (*v1.SearchResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "search request is required")
	}
	page, err := s.service.Search(ctx, domain.SearchRequest{
		Query: request.GetQuery(), EntityTypes: request.GetEntityTypes(),
		PageSize: int(request.GetPageSize()), PageToken: request.GetPageToken(),
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidRequest) {
			return nil, status.Error(codes.InvalidArgument, "search query, entity types, or page token is invalid")
		}
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.SearchResponse{Hits: make([]*v1.SearchHit, 0, len(page.Hits)), NextPageToken: page.NextPageToken}
	for _, hit := range page.Hits {
		response.Hits = append(response.Hits, &v1.SearchHit{
			EntityType: hit.EntityType, EntityId: hit.EntityID,
			Title: hit.Title, Subtitle: hit.Subtitle,
		})
	}
	return response, nil
}
