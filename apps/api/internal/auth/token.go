package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Access-token claims (ADR-0010 §3). A refresh token is never a JWT, so a
// token claiming any other typ must be rejected here.
const (
	tokenIssuer     = "carepath-api"
	tokenTypeAccess = "access"
)

// TokenIssuer mints and verifies the stateless HS256 access tokens. It is
// deliberately separate from the Repo: verification does no database work,
// which is the point of the short-lived stateless token.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret []byte, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: secret, ttl: ttl}
}

// TTL reports the configured access-token lifetime, so callers can echo the
// expiry in API responses.
func (t *TokenIssuer) TTL() time.Duration { return t.ttl }

// IssueAccessToken signs the principal into a compact JWT.
func (t *TokenIssuer) IssueAccessToken(p Principal) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(t.ttl)
	jti := make([]byte, 8)
	if _, err := rand.Read(jti); err != nil {
		return "", time.Time{}, fmt.Errorf("auth: read jti: %w", err)
	}
	claims := jwt.MapClaims{
		"iss":      tokenIssuer,
		"sub":      p.UserID,
		"username": p.Username,
		"name":     p.DisplayName,
		"roles":    p.Roles,
		"typ":      tokenTypeAccess,
		"iat":      now.Unix(),
		"exp":      expiresAt.Unix(),
		"jti":      hex.EncodeToString(jti),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseAccessToken verifies signature, issuer, and expiry, and rejects any
// token whose typ is not "access". It returns the principal the token was
// issued for.
func (t *TokenIssuer) ParseAccessToken(token string) (Principal, error) {
	parsed, err := jwt.Parse(token, func(tk *jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return Principal{}, ErrUnauthorized
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	if typ, _ := claims["typ"].(string); typ != tokenTypeAccess {
		return Principal{}, ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	name, _ := claims["name"].(string)
	roles := []string{}
	if raw, ok := claims["roles"].([]any); ok {
		for _, r := range raw {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	if sub == "" || username == "" {
		return Principal{}, ErrUnauthorized
	}
	return Principal{UserID: sub, Username: username, DisplayName: name, Roles: roles}, nil
}
