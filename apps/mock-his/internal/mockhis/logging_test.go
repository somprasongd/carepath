package mockhis

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// capturingApp wires New to a debug-level text logger writing into buf, so
// tests can assert on the exact lines the service emits (#44).
func capturingApp(t *testing.T) (*fiber.App, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	app := New(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return app, &buf
}

func post(t *testing.T, app *fiber.App, path, body string) int {
	t.Helper()
	status, _ := do(t, app, http.MethodPost, path, body)
	return status
}

// #44 AC2: every request logs method, path, status, and duration.
func TestRequestLineCarriesMethodPathStatusDuration(t *testing.T) {
	app, buf := capturingApp(t)

	if status, _ := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", ""); status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	out := buf.String()
	for _, want := range []string{
		"request completed",
		"method=GET",
		"path=/api/v1/visits/VISIT-001",
		"status=200",
		"duration_ms=",
		"request_id=",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("request log missing %q:\n%s", want, out)
		}
	}

	// Failures log the outcome status too.
	buf.Reset()
	if status, _ := do(t, app, http.MethodGet, "/api/v1/visits/NOPE", ""); status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if !strings.Contains(buf.String(), "status=404") {
		t.Fatalf("404 request not logged with its status:\n%s", buf.String())
	}
}

// #44 AC4: console-driven state changes (open visit, place order, perform,
// complete encounter, complete visit) each leave a log line identifying
// what changed.
func TestStateChangesAreLogged(t *testing.T) {
	app, buf := capturingApp(t)

	openBody := `{"patientRef":"PATIENT-X","patientName":"Test Patient","visitType":"WALKIN",` +
		`"clinics":[{"clinicCode":"MED","clinicName":"Internal Medicine"}]}`
	if s := post(t, app, "/api/v1/demo/visits", openBody); s != http.StatusOK {
		t.Fatalf("open visit = %d, want 200", s)
	}
	// The state-change attrs trail the request attrs (request_id/method/path
	// come from the request-scoped logger), so assert on the two spans.
	for _, want := range []string{
		`msg="visit opened"`,
		`visit_id=VISIT-003 patient_ref=PATIENT-X visit_type=WALKIN clinics=1 orders=0`,
	} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("open-visit log missing %q:\n%s", want, buf.String())
		}
	}

	buf.Reset()
	if s := post(t, app, "/api/v1/demo/visits/VISIT-003/orders",
		`{"orderType":"XRAY","orderName":"Chest X-Ray","orderedByClinic":"MED"}`); s != http.StatusOK {
		t.Fatalf("place order = %d, want 200", s)
	}
	if !strings.Contains(buf.String(), `visit_id=VISIT-003 order_ref=ORD-003 order_type=XRAY order_name="Chest X-Ray"`) {
		t.Fatalf("order-placed log missing:\n%s", buf.String())
	}

	buf.Reset()
	if s := post(t, app, "/api/v1/demo/orders/ORD-003/performed", ""); s != http.StatusOK {
		t.Fatalf("perform order = %d, want 200", s)
	}
	if !strings.Contains(buf.String(), `msg="order performed"`) ||
		!strings.Contains(buf.String(), `visit_id=VISIT-003 order_ref=ORD-003`) {
		t.Fatalf("order-performed log missing:\n%s", buf.String())
	}

	buf.Reset()
	if s := post(t, app, "/api/v1/demo/visits/VISIT-003/clinics/MED/complete-encounter", ""); s != http.StatusOK {
		t.Fatalf("complete encounter = %d, want 200", s)
	}
	if !strings.Contains(buf.String(), `visit_id=VISIT-003 clinic_code=MED`) {
		t.Fatalf("encounter-completed log missing:\n%s", buf.String())
	}

	buf.Reset()
	if s := post(t, app, "/api/v1/demo/visits/VISIT-003/complete", ""); s != http.StatusOK {
		t.Fatalf("complete visit = %d, want 200", s)
	}
	if !strings.Contains(buf.String(), `msg="visit completed"`) || !strings.Contains(buf.String(), `visit_id=VISIT-003`) {
		t.Fatalf("visit-completed log missing:\n%s", buf.String())
	}
}

// #44 AC4: cancelling from the console logs the visit-level state change,
// including how many open orders it took down.
func TestCancelIsLogged(t *testing.T) {
	app, buf := capturingApp(t)

	if s := post(t, app, "/api/v1/demo/visits/VISIT-001/cancel", ""); s != http.StatusOK {
		t.Fatalf("cancel = %d, want 200", s)
	}
	if !strings.Contains(buf.String(), `visit_id=VISIT-001 orders_cancelled=1`) {
		t.Fatalf("cancel log missing (seed ORD-001 should be cancelled):\n%s", buf.String())
	}
}
