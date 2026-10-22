package servicekit

import "github.com/drukleague/platform/packages/servicekit/core"

// Role names; defined in core so domain code can use them without importing gRPC.
const (
	RoleSuperAdmin    = core.RoleSuperAdmin
	RoleMatchOfficial = core.RoleMatchOfficial
	RoleTeamManager   = core.RoleTeamManager
	RoleViewer        = core.RoleViewer
	RoleService       = core.RoleService
)

// IsUserRole reports whether role can be held by a human account.
func IsUserRole(role string) bool { return core.IsUserRole(role) }

// IsUUID reports whether s is a canonical UUID.
func IsUUID(s string) bool { return core.IsUUID(s) }

// NewUUID returns a random version-4 UUID.
func NewUUID() (string, error) { return core.NewUUID() }
