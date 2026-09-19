package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		kind   Kind
		status int
	}{
		{KindInvalid, 400},
		{KindUnauthorized, 401},
		{KindForbidden, 403},
		{KindNotFound, 404},
		{KindConflict, 409},
		{KindInternal, 500},
		{KindUpstream, 502},
		{Kind(0), 500}, // unknown kinds are internal by default
	}
	for _, tt := range tests {
		if got := tt.kind.HTTPStatus(); got != tt.status {
			t.Errorf("%s.HTTPStatus() = %d, want %d", tt.kind, got, tt.status)
		}
	}
}

func TestKindOf(t *testing.T) {
	plain := errors.New("driver failure")
	domain := New(KindConflict, "duplicate code")
	wrapped := fmt.Errorf("repo: %w", domain)

	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{"nil is internal", nil, KindInternal},
		{"plain error is internal", plain, KindInternal},
		{"classified error", domain, KindConflict},
		{"classified through fmt.Errorf wrap", wrapped, KindConflict},
		{"wrapped with explicit kind", Wrapf(KindUpstream, plain, "HIS down"), KindUpstream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	if got := New(KindNotFound, "visit not found").Error(); got != "visit not found" {
		t.Errorf("New().Error() = %q", got)
	}
	cause := errors.New("dial tcp refused")
	got := Wrapf(KindUpstream, cause, "HIS unavailable").Error()
	if got != "HIS unavailable: dial tcp refused" {
		t.Errorf("Wrapf().Error() = %q", got)
	}
}

func TestUnwrapReachesCause(t *testing.T) {
	cause := errors.New("boom")
	err := Wrapf(KindInternal, cause, "context")
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is cannot reach the cause through Wrapf")
	}
}

func TestSentinelIdentitySurvivesDirectReturn(t *testing.T) {
	var ErrNotFound = New(KindNotFound, "not found")
	wrapped := fmt.Errorf("layer: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Fatal("sentinel identity lost through fmt.Errorf wrap")
	}
}
