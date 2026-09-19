package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testPrincipal = Principal{
	UserID: "user-1", Username: "somchai.r", DisplayName: "Somchai R", Roles: []string{RoleStaff},
}

func TestAccessTokenRoundTrip(t *testing.T) {
	issuer := NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), 15*time.Minute)
	token, expiresAt, err := issuer.IssueAccessToken(testPrincipal)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !time.Now().Before(expiresAt) {
		t.Fatal("expiry must be in the future")
	}
	got, err := issuer.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.UserID != testPrincipal.UserID || got.Username != testPrincipal.Username ||
		got.DisplayName != testPrincipal.DisplayName || !got.HasRole(RoleStaff) {
		t.Fatalf("round-tripped principal = %+v", got)
	}
}

func TestParseAccessTokenRejectsBadTokens(t *testing.T) {
	issuer := NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), time.Second)
	issued, _, _ := issuer.IssueAccessToken(testPrincipal)

	expired := NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), -time.Minute)
	expiredToken, _, _ := expired.IssueAccessToken(testPrincipal)

	wrongKey := NewTokenIssuer([]byte("another-secret-entirely!!!"), 15*time.Minute)
	wrongSig, _, _ := wrongKey.IssueAccessToken(testPrincipal)

	// A token with typ=refresh must never pass as an access token.
	refreshTyped, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "carepath-api", "sub": "user-1", "username": "somchai.r", "typ": "refresh",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("a-test-secret-32-bytes-long!!"))

	otherIssuer, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "someone-else", "sub": "user-1", "username": "somchai.r", "typ": "access",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("a-test-secret-32-bytes-long!!"))

	for name, token := range map[string]string{
		"empty":           "",
		"garbage":         "not-a-jwt",
		"expired":         expiredToken,
		"wrong signature": wrongSig,
		"typ refresh":     refreshTyped,
		"wrong issuer":    otherIssuer,
	} {
		if _, err := issuer.ParseAccessToken(token); err == nil {
			t.Fatalf("%s token was accepted", name)
		}
	}
	if _, err := issuer.ParseAccessToken(issued); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
}

func TestParseAccessTokenRejectsNoneAlgorithm(t *testing.T) {
	issuer := NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), time.Minute)
	// Unsigned "none"-style JWT: must fail the valid-methods check.
	unsigned, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": "carepath-api", "sub": "user-1", "username": "x", "typ": "access",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := issuer.ParseAccessToken(unsigned); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}
