package grpcadapter

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedVenueServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }
func (s *Server) CreateVenue(ctx context.Context, req *v1.CreateVenueRequest) (*v1.CreateVenueResponse, error) {
	name, district := strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetDzongkhag())
	if name == "" || district == "" {
		return nil, status.Error(codes.InvalidArgument, "venue name and dzongkhag are required")
	}
	if req.GetCapacity() < 0 {
		return nil, status.Error(codes.InvalidArgument, "capacity cannot be negative")
	}
	v := new(v1.Venue)
	err := s.db.QueryRowContext(ctx, `INSERT INTO venues(name,dzongkhag,capacity) VALUES($1,$2,$3) RETURNING id::text,name,dzongkhag,capacity`, name, district, req.GetCapacity()).Scan(&v.Id, &v.Name, &v.Dzongkhag, &v.Capacity)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not create venue")
	}
	return &v1.CreateVenueResponse{Venue: v}, nil
}
func (s *Server) GetVenue(ctx context.Context, req *v1.GetVenueRequest) (*v1.GetVenueResponse, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "venue id is required")
	}
	v := new(v1.Venue)
	err := s.db.QueryRowContext(ctx, `SELECT id::text,name,dzongkhag,capacity FROM venues WHERE id=$1`, id).Scan(&v.Id, &v.Name, &v.Dzongkhag, &v.Capacity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "venue not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "venue id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not read venue")
	}
	return &v1.GetVenueResponse{Venue: v}, nil
}
func (s *Server) ListVenues(ctx context.Context, req *v1.ListVenuesRequest) (*v1.ListVenuesResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,name,dzongkhag,capacity FROM venues WHERE ($1='' OR dzongkhag=$1) ORDER BY dzongkhag,name,id LIMIT $2 OFFSET $3`, strings.TrimSpace(req.GetDzongkhag()), size+1, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list venues")
	}
	defer rows.Close()
	result := &v1.ListVenuesResponse{Venues: make([]*v1.Venue, 0, size)}
	for rows.Next() {
		v := new(v1.Venue)
		if err := rows.Scan(&v.Id, &v.Name, &v.Dzongkhag, &v.Capacity); err != nil {
			return nil, status.Error(codes.Internal, "could not read venues")
		}
		if int32(len(result.Venues)) == size {
			result.NextPageToken = strconv.Itoa(offset + int(size))
			break
		}
		result.Venues = append(result.Venues, v)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read venues")
	}
	return result, nil
}
func page(size int32, token string) (int32, int, error) {
	if size <= 0 {
		size = 50
	}
	if size > 100 {
		size = 100
	}
	offset := 0
	if token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = n
	}
	return size, offset, nil
}
