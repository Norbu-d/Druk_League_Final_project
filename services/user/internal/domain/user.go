package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

type User struct {
	ID, Username, Role string
	CreatedAt          time.Time
}

const (
	MinPasswordLength = 10
	// MaxPasswordBytes is bcrypt's input limit; longer passwords would be silently truncated.
	MaxPasswordBytes = 72
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,40}$`)

// NormalizeUsername trims and validates a username (3-40 letters, digits, dot, dash, underscore).
func NormalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return "", core.Invalid("username must be 3 to 40 letters, digits, dots, dashes or underscores")
	}
	return username, nil
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return core.Invalid("password must be at least 10 characters")
	}
	if len(password) > MaxPasswordBytes {
		return core.Invalid("password must be at most 72 bytes")
	}
	return nil
}

// CreatableRole reports whether creatorRole may create an account with role. The admin
// creates officials, team managers and viewers; the BFF (service identity) may only create
// self-service team managers and viewers. Nobody creates another super admin through the API.
func CreatableRole(creatorRole, role string) bool {
	switch creatorRole {
	case core.RoleSuperAdmin:
		return role == core.RoleMatchOfficial || role == core.RoleTeamManager || role == core.RoleViewer
	case core.RoleService:
		return role == core.RoleTeamManager || role == core.RoleViewer
	}
	return false
}
