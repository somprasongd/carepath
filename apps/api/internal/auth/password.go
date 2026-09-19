package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP baseline for argon2id (ADR-0010 §2): memory-hard with parameters
// comfortable for the handful of logins this system sees.
const (
	argonMemoryKiB uint32 = 64 * 1024 // 64 MiB
	argonTime      uint32 = 3
	argonThreads   uint8  = 2
	argonSaltLen          = 16
	argonKeyLen           = 32
)

// dummyHash is a well-formed PHC string of an unguessable password. An
// unknown username is verified against it so response timing never reveals
// whether the account exists.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$VMHbDVxIxZfWrqX1qXQfow$/JdtEUnzrMfPuvL8tIwLWLpxHihompLfrVlTZN7PX8c"

// HashPassword derives the argon2id PHC string for a password: parameters
// travel with the hash, so they can be raised later without a flag day.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword re-derives the key using the parameters parsed out of the
// stored PHC string — never today's constants — and compares in constant
// time. Any malformed stored hash fails closed.
func VerifyPassword(password, phc string) bool {
	salt, want, params, err := parsePHC(phc)
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parsePHC(phc string) (salt, key []byte, params argonParams, err error) {
	parts := strings.Split(phc, "$")
	// Split on the leading "$" yields an empty first element: $argon2id$v=..$m=..$salt$key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, params, errors.New("auth: not an argon2id PHC string")
	}
	if _, err := fmt.Sscanf(parts[2], "v=%d", new(int)); err != nil {
		return nil, nil, params, fmt.Errorf("auth: parse version: %w", err)
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); err != nil {
		return nil, nil, params, fmt.Errorf("auth: parse parameters: %w", err)
	}
	if params.memory == 0 || params.time == 0 || params.threads == 0 {
		return nil, nil, params, errors.New("auth: zero argon2 parameter")
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return nil, nil, params, fmt.Errorf("auth: decode salt: %w", err)
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return nil, nil, params, fmt.Errorf("auth: decode key: %w", err)
	}
	if len(salt) == 0 || len(key) == 0 {
		return nil, nil, params, errors.New("auth: empty salt or key")
	}
	return salt, key, params, nil
}
