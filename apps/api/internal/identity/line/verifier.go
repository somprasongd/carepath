// Package line implements the identity.Verifier port for LINE LIFF ID
// tokens: signature (ES256, via LINE's JWKS), issuer, audience, and expiry.
package line

import (
	"context"
	"errors"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// Issuer is the fixed `iss` claim LINE sets on every ID token.
const Issuer = "https://access.line.me"

// JWKSURL is LINE's production JWKS endpoint for ID token signature verification.
const JWKSURL = "https://api.line.me/oauth2/v2.1/certs"

type claims struct {
	Name string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// Verifier validates LINE ID tokens for one LINE Login channel. It
// implements identity.Verifier.
type Verifier struct {
	channelID string
	keys      keyfunc.Keyfunc
}

// New builds a Verifier that trusts tokens issued to channelID, fetching
// (and background-refreshing) signing keys from jwksURL. Production callers
// pass JWKSURL; tests pass a fake JWKS server's URL.
func New(ctx context.Context, channelID, jwksURL string) (*Verifier, error) {
	if channelID == "" {
		return nil, errors.New("line: channelID is required")
	}
	keys, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "line: fetch JWKS")
	}
	return &Verifier{channelID: channelID, keys: keys}, nil
}

// Verify implements identity.Verifier.
func (v *Verifier) Verify(_ context.Context, idToken string) (identity.Identity, error) {
	c := &claims{}
	_, err := jwt.ParseWithClaims(idToken, c, v.keys.Keyfunc,
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(v.channelID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return identity.Identity{}, apperr.Wrapf(apperr.KindUnauthorized, err, "line: invalid ID token")
	}

	return identity.Identity{
		Source:      "line",
		ExternalID:  c.Subject,
		DisplayName: c.Name,
	}, nil
}
