// Package session owns the CarePath-issued session that stands in for a
// verified external identity (LINE LIFF today; a demo bypass for local dev).
// Other modules must depend on its Service, never its Repo.
package session

import (
	"context"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound covers a missing or expired session token: from the caller's
// side this is "not authenticated", not "resource missing".
var ErrNotFound = apperr.New(apperr.KindUnauthorized, "session not found or expired")

// ErrDemoAuthDisabled is returned for a demo-source request when the backend
// gate (ALLOW_DEMO_AUTH) is off.
var ErrDemoAuthDisabled = apperr.New(apperr.KindUnauthorized, "demo auth is disabled")

// ErrLineAuthNotConfigured is returned for a "line" source request when the
// server was started without LINE_CHANNEL_ID — a deployment/config problem,
// not a bad request from the caller.
var ErrLineAuthNotConfigured = apperr.New(apperr.KindInternal, "line auth is not configured")

// Session is a CarePath-issued credential standing in for a verified identity.
type Session struct {
	Token     string
	Identity  identity.Identity
	ExpiresAt time.Time
}

// Repo is the persistence port of this module.
type Repo interface {
	Create(ctx context.Context, s Session) error
	// Get returns ErrNotFound for a token with no row, or one whose row has
	// already expired — callers never need to check ExpiresAt themselves.
	Get(ctx context.Context, token string) (Session, error)
}
