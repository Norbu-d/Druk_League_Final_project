package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/drukleague/platform/services/search/internal/domain"
	"github.com/drukleague/platform/services/search/internal/ports"
)

const (
	defaultPageSize = 20
	maximumPageSize = 100
	maximumQueryLen = 200
	maximumOffset   = 1_000_000
)

var ErrInvalidRequest = errors.New("invalid search request")

var validEntityTypes = map[string]struct{}{
	"team": {}, "player": {}, "tournament": {},
}

type Service struct{ repository ports.Repository }

func New(repository ports.Repository) *Service { return &Service{repository: repository} }

func (s *Service) Search(ctx context.Context, request domain.SearchRequest) (domain.Page, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" || utf8.RuneCountInString(query) > maximumQueryLen {
		return domain.Page{}, ErrInvalidRequest
	}

	entityTypes := make([]string, 0, len(request.EntityTypes))
	seen := make(map[string]struct{}, len(request.EntityTypes))
	for _, entityType := range request.EntityTypes {
		entityType = strings.ToLower(strings.TrimSpace(entityType))
		if _, valid := validEntityTypes[entityType]; !valid {
			return domain.Page{}, ErrInvalidRequest
		}
		if _, exists := seen[entityType]; exists {
			continue
		}
		seen[entityType] = struct{}{}
		entityTypes = append(entityTypes, entityType)
	}

	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	} else if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	offset, err := parseOffset(request.PageToken)
	if err != nil || offset > maximumOffset {
		return domain.Page{}, ErrInvalidRequest
	}
	if offset > maximumOffset-pageSize {
		return domain.Page{}, ErrInvalidRequest
	}

	hits, err := s.repository.Search(ctx, query, entityTypes, pageSize+1, offset)
	if err != nil {
		return domain.Page{}, fmt.Errorf("search documents: %w", err)
	}
	page := domain.Page{Hits: hits}
	if len(hits) > pageSize {
		page.Hits = hits[:pageSize]
		page.NextPageToken = strconv.Itoa(offset + pageSize)
	}
	return page, nil
}

func parseOffset(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, ErrInvalidRequest
	}
	return offset, nil
}
