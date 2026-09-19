package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
)

// TestErrorStatusPerKind verifies the full domain-error → HTTP mapping, that
// client-safe messages are exposed, that internal details are masked, and
// that every response carries the machine-readable kind as code (#28).
func TestErrorStatusPerKind(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		wantInBody string
		notInBody  string
	}{
		{"invalid → 400", apperr.New(apperr.KindInvalid, "visitId is required"), 400, "invalid", "visitId is required", ""},
		{"unauthorized → 401", apperr.New(apperr.KindUnauthorized, "missing token"), 401, "unauthorized", "missing token", ""},
		{"forbidden → 403", apperr.New(apperr.KindForbidden, "staff role required"), 403, "forbidden", "staff role required", ""},
		{"not found → 404", apperr.New(apperr.KindNotFound, "visit not found"), 404, "not_found", "visit not found", ""},
		{"conflict → 409", apperr.New(apperr.KindConflict, "service point code already exists"), 409, "conflict", "service point code already exists", ""},
		{
			"upstream → 502 masked",
			apperr.Wrapf(apperr.KindUpstream, errors.New("dial tcp [::1]:8090: connect: connection refused"), "HIS unavailable"),
			502,
			"upstream",
			"upstream service unavailable",
			"connection refused",
		},
		{
			"internal → 500 masked",
			apperr.Wrapf(apperr.KindInternal, errors.New("pq: connection reset"), "servicepoint: get by code"),
			500,
			"internal",
			"internal server error",
			"pq:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/fail", func(c fiber.Ctx) error { return Error(c, tt.err) })

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/fail", nil))
			if err != nil {
				t.Fatalf("test request: %v", err)
			}
			if resp.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var envelope ErrorResponse
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatalf("decode body %q: %v", body, err)
			}
			if envelope.Code != tt.code {
				t.Fatalf("code = %q, want %q", envelope.Code, tt.code)
			}
			if !strings.Contains(envelope.Error, tt.wantInBody) {
				t.Fatalf("body = %q, want it to contain %q", envelope.Error, tt.wantInBody)
			}
			if tt.notInBody != "" && strings.Contains(envelope.Error, tt.notInBody) {
				t.Fatalf("body = %q leaked internal detail %q", envelope.Error, tt.notInBody)
			}
		})
	}

	if !strings.Contains(buf.String(), "pq: connection reset") {
		t.Fatalf("internal cause missing from logs:\n%s", buf.String())
	}
}

func TestPlainErrorMapsTo500(t *testing.T) {
	app := fiber.New()
	app.Get("/fail", func(c fiber.Ctx) error { return Error(c, errors.New("surprise")) })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/fail", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var envelope ErrorResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if envelope.Code != "internal" {
		t.Fatalf("code = %q, want internal", envelope.Code)
	}
}
