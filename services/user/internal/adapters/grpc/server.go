package grpcadapter

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedUserServiceServer
	db *sql.DB
}

func New(db *sql.DB) *Server { return &Server{db: db} }

func (s *Server) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	if username == "" || len(req.GetPassword()) < 10 {
		return nil, status.Error(codes.InvalidArgument, "username is required and password must be at least 10 characters")
	}
	role := req.GetRole()
	if role != "match_official" && role != "viewer" {
		return nil, status.Error(codes.InvalidArgument, "role must be match_official or viewer")
	}
	hash, err := hashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, "could not secure password")
	}
	var user v1.User
	var created time.Time
	err = s.db.QueryRowContext(ctx, `INSERT INTO users(username,password_hash,role) VALUES($1,$2,$3) RETURNING id::text,username,role,created_at`, username, hash, role).Scan(&user.Id, &user.Username, &user.Role, &created)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		}
		return nil, status.Error(codes.Internal, "could not create user")
	}
	user.CreatedAt = timestamppb.New(created)
	return &v1.CreateUserResponse{User: &user}, nil
}

func (s *Server) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	var user v1.User
	var created time.Time
	err := s.db.QueryRowContext(ctx, `SELECT id::text,username,role,created_at FROM users WHERE id=$1`, id).Scan(&user.Id, &user.Username, &user.Role, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not read user")
	}
	user.CreatedAt = timestamppb.New(created)
	return &v1.GetUserResponse{User: &user}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	pageSize, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	role := req.GetRole()
	if role != "" && role != "super_admin" && role != "match_official" && role != "viewer" {
		return nil, status.Error(codes.InvalidArgument, "unsupported role filter")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,username,role,created_at FROM users WHERE ($1='' OR role=$1) ORDER BY username LIMIT $2 OFFSET $3`, role, pageSize+1, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not list users")
	}
	defer rows.Close()
	result := &v1.ListUsersResponse{Users: make([]*v1.User, 0, pageSize)}
	for rows.Next() {
		var user v1.User
		var created time.Time
		if err := rows.Scan(&user.Id, &user.Username, &user.Role, &created); err != nil {
			return nil, status.Error(codes.Internal, "could not read users")
		}
		if int32(len(result.Users)) == pageSize {
			result.NextPageToken = strconv.Itoa(offset + int(pageSize))
			break
		}
		user.CreatedAt = timestamppb.New(created)
		result.Users = append(result.Users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "could not read users")
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

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const iterations = 120000
	derived := pbkdf2([]byte(password), salt, iterations, 32)
	return fmt.Sprintf("pbkdf2$%d$%s$%s", iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	out := make([]byte, 0, keyLen)
	for block := uint32(1); len(out) < keyLen; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
