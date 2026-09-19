// Package auth owns staff/admin identity: users with argon2id password
// hashes, role-based access, and the JWT access / opaque refresh token pair
// (ADR-0010). It is deliberately separate from the patient session module —
// a nurse has no LINE identity and a patient has no password. Other modules
// depend on the Service and the middleware here, never on the Repo.
package auth

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// Role codes enforced by the MVP (ADR-0010 §5). Splitting STAFF or adding
// EXECUTIVE later is a data change, not a schema change.
const (
	RoleAdmin = "ADMIN"
	RoleStaff = "STAFF"
)

// ErrInvalidCredentials covers every login failure — unknown username, wrong
// password, deactivated account. Callers must not distinguish them.
var ErrInvalidCredentials = apperr.New(apperr.KindUnauthorized, "invalid credentials")

// ErrUnauthorized is a missing, malformed, or expired access token.
var ErrUnauthorized = apperr.New(apperr.KindUnauthorized, "missing or invalid access token")

// ErrForbidden is a valid token whose roles do not include a required one.
var ErrForbidden = apperr.New(apperr.KindForbidden, "insufficient role")

// User is a staff/admin account as stored, hash included. It never leaves
// this module — the outside world sees Principal.
type User struct {
	UserID       string
	Username     string
	PasswordHash string
	FullName     string
	IsActive     bool
	Roles        []string
}

// HasRole reports whether the account holds any of the given role codes.
func (u User) HasRole(roles ...string) bool {
	for _, want := range roles {
		for _, have := range u.Roles {
			if have == want {
				return true
			}
		}
	}
	return false
}

// Principal is the resolved acting user carried in request contexts after
// token verification: enough for logs, the UI header, and the audit trail.
type Principal struct {
	UserID      string
	Username    string
	DisplayName string
	Roles       []string
}

// HasRole reports whether the principal holds any of the given role codes.
func (p Principal) HasRole(roles ...string) bool {
	for _, want := range roles {
		for _, have := range p.Roles {
			if have == want {
				return true
			}
		}
	}
	return false
}

// RefreshToken is the persisted state of one opaque refresh token. The raw
// token exists only in the client and in the return value of the service
// call that minted it; persistence sees TokenHash.
type RefreshToken struct {
	TokenHash string
	UserID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
}

// Usable reports whether the row can still be exchanged for a new pair.
func (t RefreshToken) Usable(now time.Time) bool {
	return t.UsedAt == nil && t.RevokedAt == nil && t.ExpiresAt.After(now)
}

// Repo is the persistence port of this module.
type Repo interface {
	// GetUserByUsername returns ErrInvalidCredentials-shaped not-found: the
	// service layer normalizes every lookup failure the same way.
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)
	// MarkRefreshTokenUsed atomically spends a token: true when the row was
	// usable and is now marked used, false when another rotation got there
	// first (treated as reuse).
	MarkRefreshTokenUsed(ctx context.Context, tokenHash string, at time.Time) (bool, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error
	RevokeUserRefreshTokens(ctx context.Context, userID string, at time.Time) error
}
