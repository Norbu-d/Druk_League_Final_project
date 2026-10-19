package domain

import "time"

type User struct{ ID, Username, Role string }

// Session is one signed-in device. Only the SHA-256 of its refresh token is stored.
type Session struct {
	ID, UserID string
	ExpiresAt  time.Time
	Revoked    bool
}

// Tokens is what a successful login or refresh returns.
type Tokens struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
	SessionID       string
	User            User
}

// Attempts tracks failed logins for one username.
type Attempts struct {
	Failed      int
	WindowStart time.Time
	LockedUntil time.Time
}

// LockoutPolicy limits password guessing: MaxFailures inside Window locks the username for LockFor.
type LockoutPolicy struct {
	MaxFailures int
	Window      time.Duration
	LockFor     time.Duration
}

var DefaultLockout = LockoutPolicy{MaxFailures: 5, Window: 15 * time.Minute, LockFor: 15 * time.Minute}

// Locked reports whether the username is currently locked.
func (p LockoutPolicy) Locked(a Attempts, now time.Time) bool {
	return now.Before(a.LockedUntil)
}

// AfterFailure returns the attempt record after one more failed login.
func (p LockoutPolicy) AfterFailure(a Attempts, now time.Time) Attempts {
	if a.WindowStart.IsZero() || now.Sub(a.WindowStart) > p.Window {
		a = Attempts{WindowStart: now}
	}
	a.Failed++
	if a.Failed >= p.MaxFailures {
		a.LockedUntil = now.Add(p.LockFor)
		a.Failed, a.WindowStart = 0, now
	}
	return a
}
