// Package identity defines the outbound port for verifying an external
// identity assertion before CarePath trusts it. The concrete adapters (line,
// eventually demo) live in their own subpackages, matching internal/his's
// port + adapter split — this package never talks to a specific provider.
package identity

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrInvalidToken is returned when an identity assertion fails verification
// (bad signature, expired, wrong issuer/audience, malformed).
var ErrInvalidToken = apperr.New(apperr.KindUnauthorized, "invalid identity token")

// Identity is the subset of a verified external identity CarePath trusts.
type Identity struct {
	Source      string
	ExternalID  string
	DisplayName string
}

// Verifier checks an identity assertion (e.g. a LINE LIFF ID token) and
// returns the identity it asserts. Callers must treat any error as "not
// trusted" — there is no partial verification.
type Verifier interface {
	Verify(ctx context.Context, idToken string) (Identity, error)
}
