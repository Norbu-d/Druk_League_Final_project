package servicekit

import (
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToStatus(t *testing.T) {
	if ToStatus(nil) != nil {
		t.Fatal("nil must stay nil")
	}
	for err, code := range map[error]codes.Code{
		core.Invalid("name is required"): codes.InvalidArgument, core.NotFound("team"): codes.NotFound,
		core.Conflict("dup"): codes.AlreadyExists, core.Forbidden("no"): codes.PermissionDenied,
		core.ErrUnauthenticated: codes.Unauthenticated, core.Dependency("team"): codes.Unavailable,
		core.Precondition("closed"): codes.FailedPrecondition, errors.New("pq: secret detail"): codes.Internal,
		status.Error(codes.ResourceExhausted, "slow down"): codes.ResourceExhausted,
	} {
		got := ToStatus(err)
		if status.Code(got) != code {
			t.Fatalf("%v: got %s want %s", err, status.Code(got), code)
		}
		if code == codes.Internal && status.Convert(got).Message() != "internal error" {
			t.Fatal("internal details leaked")
		}
	}
	if status.Convert(ToStatus(core.Invalid("name is required"))).Message() != "invalid request: name is required" {
		t.Fatal("detail lost")
	}
}

func TestFromStatus(t *testing.T) {
	if FromStatus(nil, "team") != nil {
		t.Fatal("nil must stay nil")
	}
	for code, want := range map[codes.Code]error{
		codes.NotFound: core.ErrNotFound, codes.InvalidArgument: core.ErrInvalid, codes.PermissionDenied: core.ErrForbidden,
		codes.Unauthenticated: core.ErrUnauthenticated, codes.Unavailable: core.ErrDependency, codes.Internal: core.ErrDependency,
	} {
		if got := FromStatus(status.Error(code, "x"), "team"); !errors.Is(got, want) {
			t.Fatalf("%s: got %v", code, got)
		}
	}
}

func TestCaller(t *testing.T) {
	if _, err := Caller(Identity{}, false); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("anonymous accepted")
	}
	if identity, err := Caller(Identity{ID: "u"}, true); err != nil || identity.ID != "u" {
		t.Fatal("identity lost")
	}
}
