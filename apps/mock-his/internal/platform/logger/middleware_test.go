package logger

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func newTestApp(t *testing.T) (*fiber.App, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	base := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(base)

	app := fiber.New()
	app.Use(Middleware(base))
	return app, &buf
}

func TestMiddlewareAttachesRequestScopedLogger(t *testing.T) {
	app, buf := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		FromContext(c.Context()).Info("inside handler")
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	out := buf.String()
	if !strings.Contains(out, "inside handler") {
		t.Fatalf("handler log missing request-scoped logger:\n%s", out)
	}
	if !strings.Contains(out, "request_id=") {
		t.Fatalf("logs missing request_id:\n%s", out)
	}
	if !strings.Contains(out, "request completed") || !strings.Contains(out, "status=200") {
		t.Fatalf("missing completion line with status:\n%s", out)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID response header not set")
	}
}

func TestMiddlewareHonorsIncomingRequestID(t *testing.T) {
	app, buf := newTestApp(t)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "trace-abc-123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}

	if got := resp.Header.Get("X-Request-ID"); got != "trace-abc-123" {
		t.Fatalf("X-Request-ID = %q, want incoming trace-abc-123", got)
	}
	if !strings.Contains(buf.String(), "request_id=trace-abc-123") {
		t.Fatalf("logs missing incoming request id:\n%s", buf.String())
	}
}

func TestMiddlewareRecoversPanic(t *testing.T) {
	app, buf := newTestApp(t)
	app.Get("/boom", func(c fiber.Ctx) error {
		panic("kaboom")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if !strings.Contains(buf.String(), "panic recovered") || !strings.Contains(buf.String(), "kaboom") {
		t.Fatalf("missing panic log:\n%s", buf.String())
	}
}

func TestFromContextFallsBackToDefault(t *testing.T) {
	if FromContext(context.Background()) == nil {
		t.Fatal("FromContext returned nil for empty ctx")
	}
	log := slog.New(slog.DiscardHandler)
	ctx := IntoContext(context.Background(), log)
	if FromContext(ctx) != log {
		t.Fatal("FromContext did not return the logger stored in ctx")
	}
}
