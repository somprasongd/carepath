package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash is not the OWASP-baseline PHC string: %s", hash)
	}
	if !VerifyPassword("s3cret!", hash) {
		t.Fatal("correct password failed to verify")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("wrong password verified")
	}
}

func TestSamePasswordHashesDiffer(t *testing.T) {
	a, _ := HashPassword("demo")
	b, _ := HashPassword("demo")
	if a == b {
		t.Fatal("two hashes of the same password are identical — per-user salt missing")
	}
	if !VerifyPassword("demo", a) || !VerifyPassword("demo", b) {
		t.Fatal("both hashes must verify against the same password")
	}
}

func TestVerifyPasswordRejectsTamperedPHC(t *testing.T) {
	hash, _ := HashPassword("demo")
	cases := map[string]string{
		"not a phc":            "plaintext",
		"wrong algorithm":      strings.Replace(hash, "argon2id", "argon2i", 1),
		"tampered key":         hash[:len(hash)-2] + "AA",
		"truncated":            hash[:len(hash)/2],
		"zero parameters":      "$argon2id$v=19$m=0,t=0,p=0$6cQYsXhfiy53rxDIgSsMNA$YZ4Y+NsB9qU90YiJ3Yu9dZ7",
		"empty salt and key":   "$argon2id$v=19$m=65536,t=3,p=2$$",
		"undecodable base64[]": "$argon2id$v=19$m=65536,t=3,p=2$???$YZ4Y+NsB9qU90YiJ3Yu9dZ7",
	}
	for name, phc := range cases {
		if VerifyPassword("demo", phc) {
			t.Fatalf("%s: tampered PHC string verified", name)
		}
	}
}

// The migration seeds two users whose committed hashes must keep verifying
// against "demo", or every fresh environment cannot log in.
func TestSeedHashesVerifyDemoPassword(t *testing.T) {
	admin := "$argon2id$v=19$m=65536,t=3,p=2$6cQYsXhfiy53rxDIgSsMNA$YZ4Y+NsB9qU90YiJ3Yu9dZ7/HCCyjXwJ6Q6nLjmxWI0"
	staff := "$argon2id$v=19$m=65536,t=3,p=2$edCrKYCnF4fd9sVqET6Aeg$7pBauHGQylj7XN9uRqG9smLSntIe0qr8DA67hiPu3dk"
	if !VerifyPassword("demo", admin) || !VerifyPassword("demo", staff) {
		t.Fatal("committed seed hashes no longer verify against the demo password")
	}
	if !VerifyPassword("demo", dummyHash) {
		t.Fatal("the timing-equalizing dummy hash must also verify against its own password")
	}
}
