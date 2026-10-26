package core

import (
	"errors"
	"fmt"
)

// Sentinel errors shared by domain and app layers. Adapters map them to transport codes
// (servicekit.ToStatus); wrap them with a detail using the helpers below.
var (
	ErrInvalid         = errors.New("invalid request")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrForbidden       = errors.New("permission denied")
	ErrUnauthenticated = errors.New("authentication required")
	ErrDependency      = errors.New("dependent service unavailable")
	ErrPrecondition    = errors.New("failed precondition")
)

func Invalid(detail string) error      { return fmt.Errorf("%w: %s", ErrInvalid, detail) }
func NotFound(detail string) error     { return fmt.Errorf("%w: %s", ErrNotFound, detail) }
func Conflict(detail string) error     { return fmt.Errorf("%w: %s", ErrConflict, detail) }
func Forbidden(detail string) error    { return fmt.Errorf("%w: %s", ErrForbidden, detail) }
func Dependency(detail string) error   { return fmt.Errorf("%w: %s", ErrDependency, detail) }
func Precondition(detail string) error { return fmt.Errorf("%w: %s", ErrPrecondition, detail) }
