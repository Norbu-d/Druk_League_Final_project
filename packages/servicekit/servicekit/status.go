package servicekit

import (
	"errors"
	"log"

	"github.com/drukleague/platform/packages/servicekit/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus maps core sentinel errors to gRPC status codes. The wrapped message of a known
// error is returned to the caller; anything else is logged and reported as Internal so
// database or driver details never leak.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok && !isCoreError(err) {
		return err
	}
	switch {
	case errors.Is(err, core.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, core.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, core.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, core.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, core.ErrDependency):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, core.ErrPrecondition):
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	log.Printf("internal error: %v", err)
	return status.Error(codes.Internal, "internal error")
}

func isCoreError(err error) bool {
	for _, sentinel := range []error{core.ErrInvalid, core.ErrNotFound, core.ErrConflict, core.ErrForbidden, core.ErrUnauthenticated, core.ErrDependency, core.ErrPrecondition} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}

// FromStatus maps a gRPC error from a dependency back to core errors, so app layers can
// distinguish "the referenced thing does not exist" from "the dependency is down".
func FromStatus(err error, what string) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.NotFound:
		return core.NotFound(what + " not found")
	case codes.InvalidArgument:
		return core.Invalid(what + " is invalid")
	case codes.PermissionDenied:
		return core.Forbidden("not allowed to read " + what)
	case codes.Unauthenticated:
		return core.ErrUnauthenticated
	}
	return core.Dependency("could not reach the service holding " + what)
}

// Caller returns the authenticated identity or core.ErrUnauthenticated.
func Caller(ctxIdentity Identity, ok bool) (Identity, error) {
	if !ok || ctxIdentity.ID == "" {
		return Identity{}, core.ErrUnauthenticated
	}
	return ctxIdentity, nil
}
