package line

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testChannelID = "test-channel-id"

// jwksServer serves a JWKS containing exactly one EC public key under kid.
func jwksServer(t *testing.T, kid string, pub *ecdsa.PublicKey) *httptest.Server {
	t.Helper()

	// Uncompressed SEC1 point: 0x04 || X || Y, each field-sized (32 bytes for P-256).
	raw, err := pub.Bytes()
	if err != nil {
		t.Fatalf("encode public key: %v", err)
	}
	fieldSize := (len(raw) - 1) / 2
	jwk := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"kid": kid,
		"use": "sig",
		"alg": "ES256",
		"x":   base64.RawURLEncoding.EncodeToString(raw[1 : 1+fieldSize]),
		"y":   base64.RawURLEncoding.EncodeToString(raw[1+fieldSize:]),
	}
	body, err := json.Marshal(map[string]any{"keys": []any{jwk}})
	if err != nil {
		t.Fatalf("marshal JWKS: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func signToken(t *testing.T, key *ecdsa.PrivateKey, kid string, c claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func baseClaims(aud string, expiresAt time.Time) claims {
	return claims{
		Name: "Somchai Patient",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   "U1234567890abcdef",
			Audience:  jwt.ClaimStrings{aud},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}
}

func newTestVerifier(t *testing.T, jwksURL string) *Verifier {
	t.Helper()
	v, err := New(context.Background(), testChannelID, jwksURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func TestVerifierValidToken(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv := jwksServer(t, "test-kid", &key.PublicKey)
	v := newTestVerifier(t, srv.URL)

	token := signToken(t, key, "test-kid", baseClaims(testChannelID, time.Now().Add(time.Hour)))

	got, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify() unexpected error: %v", err)
	}
	if got.Source != "line" {
		t.Errorf("Source = %q, want line", got.Source)
	}
	if got.ExternalID != "U1234567890abcdef" {
		t.Errorf("ExternalID = %q, want U1234567890abcdef", got.ExternalID)
	}
	if got.DisplayName != "Somchai Patient" {
		t.Errorf("DisplayName = %q, want Somchai Patient", got.DisplayName)
	}
}

func TestVerifierExpiredToken(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv := jwksServer(t, "test-kid", &key.PublicKey)
	v := newTestVerifier(t, srv.URL)

	token := signToken(t, key, "test-kid", baseClaims(testChannelID, time.Now().Add(-time.Hour)))

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("Verify() expected error for expired token, got nil")
	}
}

func TestVerifierWrongAudience(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv := jwksServer(t, "test-kid", &key.PublicKey)
	v := newTestVerifier(t, srv.URL)

	token := signToken(t, key, "test-kid", baseClaims("some-other-channel", time.Now().Add(time.Hour)))

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("Verify() expected error for wrong audience, got nil")
	}
}

func TestVerifierBadSignature(t *testing.T) {
	realKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	forgedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate forged key: %v", err)
	}
	srv := jwksServer(t, "test-kid", &realKey.PublicKey)
	v := newTestVerifier(t, srv.URL)

	// Signed with forgedKey but claims a kid whose published key is realKey's public key.
	token := signToken(t, forgedKey, "test-kid", baseClaims(testChannelID, time.Now().Add(time.Hour)))

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("Verify() expected error for forged signature, got nil")
	}
}
