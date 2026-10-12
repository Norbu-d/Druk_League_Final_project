package grpcadapter

import (
	"context"
	"database/sql"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedOfficialServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }
func (s *Server) AssignOfficial(ctx context.Context, req *v1.AssignOfficialRequest) (*v1.AssignOfficialResponse, error) {
	matchID, userID := strings.TrimSpace(req.GetMatchId()), strings.TrimSpace(req.GetOfficialUserId())
	if matchID == "" || userID == "" {
		return nil, status.Error(codes.InvalidArgument, "match id and official user id are required")
	}
	a := new(v1.OfficialAssignment)
	var assigned time.Time
	err := s.db.QueryRowContext(ctx, `INSERT INTO official_assignments(match_id,official_user_id) VALUES($1,$2) ON CONFLICT(match_id) DO UPDATE SET official_user_id=EXCLUDED.official_user_id,assigned_at=now() RETURNING id::text,match_id::text,official_user_id::text,assigned_at`, matchID, userID).Scan(&a.Id, &a.MatchId, &a.OfficialUserId, &assigned)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "match and user ids must be UUIDs")
		}
		return nil, status.Error(codes.Internal, "could not assign official")
	}
	a.AssignedAt = timestamppb.New(assigned)
	return &v1.AssignOfficialResponse{Assignment: a}, nil
}
func (s *Server) ListAssignments(ctx context.Context, req *v1.ListAssignmentsRequest) (*v1.ListAssignmentsResponse, error) {
	if strings.TrimSpace(req.GetTournamentId()) != "" {
		return nil, status.Error(codes.FailedPrecondition, "tournament filtering requires Match service integration")
	}
	userID := strings.TrimSpace(req.GetOfficialUserId())
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,match_id::text,official_user_id::text,assigned_at FROM official_assignments WHERE ($1='' OR official_user_id::text=$1) ORDER BY assigned_at DESC,id`, userID)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return nil, status.Error(codes.InvalidArgument, "official user id must be a UUID")
		}
		return nil, status.Error(codes.Internal, "could not list assignments")
	}
	defer rows.Close()
	result := &v1.ListAssignmentsResponse{Assignments: make([]*v1.OfficialAssignment, 0)}
	for rows.Next() {
		a := new(v1.OfficialAssignment)
		var assigned time.Time
		if err := rows.Scan(&a.Id, &a.MatchId, &a.OfficialUserId, &assigned); err != nil {
			return nil, status.Error(codes.Internal, "could not read assignments")
		}
		a.AssignedAt = timestamppb.New(assigned)
		result.Assignments = append(result.Assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read assignments")
	}
	return result, nil
}
