package mockhis

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// do runs one request against the app and decodes the JSON response body.
func do(t *testing.T, app *fiber.App, method, path, body string) (int, map[string]any) {
	t.Helper()
	status, raw := doRaw(t, app, method, path, body)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: decode response %q: %v", method, path, raw, err)
	}
	return status, out
}

// doList is do for endpoints whose response body is a JSON array.
func doList(t *testing.T, app *fiber.App, method, path, body string) (int, []any) {
	t.Helper()
	status, raw := doRaw(t, app, method, path, body)
	var out []any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: decode response %q: %v", method, path, raw, err)
	}
	return status, out
}

func doRaw(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req, _ = http.NewRequest(method, path, nil)
	} else {
		req, _ = http.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read response: %v", method, path, err)
	}
	return resp.StatusCode, raw
}

func eventTypes(body map[string]any) []string {
	var types []string
	for _, raw := range body["events"].([]any) {
		types = append(types, raw.(map[string]any)["type"].(string))
	}
	return types
}

func TestVisitSnapshotMatchesContract(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	if body["visitId"] != "VISIT-001" || body["patientRef"] != "PATIENT-DEMO-001" || body["status"] != "ACTIVE" {
		t.Fatalf("snapshot header = %v, want VISIT-001 / PATIENT-DEMO-001 / ACTIVE", body)
	}
	if body["visitType"] != "APPOINTMENT" {
		t.Fatalf("visitType = %v, want APPOINTMENT", body["visitType"])
	}
	clinics := body["clinics"].([]any)
	if len(clinics) != 1 || clinics[0].(map[string]any)["code"] != "MED" {
		t.Fatalf("clinics = %v, want [{code: MED}]", clinics)
	}
	orders := body["orders"].([]any)
	if len(orders) != 1 || orders[0].(map[string]any)["orderType"] != "LAB" || orders[0].(map[string]any)["status"] != "PLACED" {
		t.Fatalf("orders = %v, want one PLACED LAB order", orders)
	}

	if status, _ := do(t, app, http.MethodGet, "/api/v1/visits/NOPE", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}
}

func TestOpenVisit(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits",
		`{"visitType":"WALKIN","patientRef":"HN-X","patientName":"ทดสอบ",`+
			`"clinics":[{"clinicCode":"SURG"}]}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	if body["visitId"] != "VISIT-002" || body["patientRef"] != "HN-X" || body["status"] != "ACTIVE" {
		t.Fatalf("opened visit = %v, want VISIT-002 / HN-X / ACTIVE", body)
	}
	clinics := body["clinics"].([]any)
	if len(clinics) != 1 || clinics[0].(map[string]any)["code"] != "SURG" {
		t.Fatalf("clinics = %v, want [{code: SURG}]", clinics)
	}

	// Defaults: no patientRef -> a demo id is generated.
	status, body = do(t, app, http.MethodPost, "/api/v1/demo/visits",
		`{"visitType":"APPOINTMENT","clinics":[{"clinicCode":"MED"}]}`)
	if status != http.StatusOK || body["visitId"] != "VISIT-003" {
		t.Fatalf("second open = %d %v, want 200 VISIT-003", status, body)
	}
	if body["patientRef"] != "PATIENT-DEMO-003" {
		t.Fatalf("generated patientRef = %v, want PATIENT-DEMO-003", body["patientRef"])
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits", `not json`); status != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"visitType":"WALKIN","clinics":[]}`); status != http.StatusBadRequest {
		t.Fatalf("no clinics status = %d, want 400", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"visitType":"BOGUS","clinics":[{"clinicCode":"MED"}]}`); status != http.StatusBadRequest {
		t.Fatalf("bad visitType status = %d, want 400", status)
	}
}

func TestOpenVisitWithPreVisitOrders(t *testing.T) {
	app := New()
	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits",
		`{"visitType":"APPOINTMENT","clinics":[{"clinicCode":"MED"}],`+
			`"orders":[{"orderType":"LAB","orderName":"CBC","orderedByClinic":"MED"}]}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	orders := body["orders"].([]any)
	if len(orders) != 1 || orders[0].(map[string]any)["orderType"] != "LAB" {
		t.Fatalf("pre-visit orders = %v, want one LAB order", orders)
	}
}

func TestAddClinic(t *testing.T) {
	app := New()
	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/clinics", `{"clinicCode":"SURG"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	clinics := body["clinics"].([]any)
	if len(clinics) != 2 || clinics[1].(map[string]any)["code"] != "SURG" {
		t.Fatalf("clinics = %v, want MED then SURG", clinics)
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/clinics", `{"clinicCode":"SURG"}`); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/clinics", `{"clinicCode":""}`); status != http.StatusBadRequest {
		t.Fatalf("blank clinic code status = %d, want 400", status)
	}
}

func TestOrderLifecycle(t *testing.T) {
	app := New()
	status, order := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"orderType":"XRAY","orderName":"Chest X-Ray","orderedByClinic":"MED"}`)
	if status != http.StatusOK || order["status"] != "PLACED" {
		t.Fatalf("place order = %d %v, want 200 PLACED", status, order)
	}
	ref := order["orderRef"].(string)

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/resulted", ""); status != http.StatusConflict {
		t.Fatalf("resulted before performed = %d, want 409", status)
	}

	status, order = do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/performed", "")
	if status != http.StatusOK || order["status"] != "PERFORMED" {
		t.Fatalf("mark performed = %d %v, want 200 PERFORMED", status, order)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/performed", ""); status != http.StatusConflict {
		t.Fatalf("double performed = %d, want 409", status)
	}

	status, order = do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/resulted", "")
	if status != http.StatusOK || order["status"] != "RESULTED" {
		t.Fatalf("mark resulted = %d %v, want 200 RESULTED", status, order)
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/cancel", ""); status != http.StatusConflict {
		t.Fatalf("cancel resulted order = %d, want 409", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/orders/NOPE/performed", ""); status != http.StatusNotFound {
		t.Fatalf("unknown order = %d, want 404", status)
	}
}

func TestOrderCancel(t *testing.T) {
	app := New()
	_, order := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"orderType":"EKG","orderName":"ECG","orderedByClinic":"MED"}`)
	ref := order["orderRef"].(string)

	status, cancelled := do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/cancel", "")
	if status != http.StatusOK || cancelled["status"] != "CANCELLED" {
		t.Fatalf("cancel = %d %v, want 200 CANCELLED", status, cancelled)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/cancel", ""); status != http.StatusConflict {
		t.Fatalf("double cancel = %d, want 409", status)
	}
}

func TestPlaceOrderRejectsUnknownTypeAndFinishedVisit(t *testing.T) {
	app := New()
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"orderType":"MRI","orderName":"MRI","orderedByClinic":"MED"}`); status != http.StatusBadRequest {
		t.Fatalf("unknown order type status = %d, want 400", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/orders",
		`{"orderType":"LAB","orderName":"CBC","orderedByClinic":"MED"}`); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}

	do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/complete", "")
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"orderType":"LAB","orderName":"CBC","orderedByClinic":"MED"}`); status != http.StatusConflict {
		t.Fatalf("order on completed visit status = %d, want 409", status)
	}
}

func TestCompleteEncounter(t *testing.T) {
	app := New()
	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/clinics/MED/complete-encounter", "")
	if status != http.StatusOK || body["visitId"] != "VISIT-001" {
		t.Fatalf("complete-encounter = %d %v, want 200 VISIT-001", status, body)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/clinics/SURG/complete-encounter", ""); status != http.StatusNotFound {
		t.Fatalf("unassigned clinic status = %d, want 404", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/clinics/MED/complete-encounter", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}
}

func TestCompleteVisit(t *testing.T) {
	app := New()
	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/complete", "")
	if status != http.StatusOK || body["status"] != "COMPLETED" {
		t.Fatalf("complete = %d %v, want 200 COMPLETED", status, body)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/complete", ""); status != http.StatusConflict {
		t.Fatalf("double complete = %d, want 409", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/complete", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit complete = %d, want 404", status)
	}
}

func TestEventFeedCursorAndEnvelope(t *testing.T) {
	app := New()

	status, page := do(t, app, http.MethodGet, "/api/v1/events?limit=1", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	events := page["events"].([]any)
	if len(events) != 1 || page["nextAfter"] != "EVT-000001" {
		t.Fatalf("first page = %v nextAfter %v, want 1 event and EVT-000001", len(events), page["nextAfter"])
	}
	first := events[0].(map[string]any)
	for _, key := range []string{"eventId", "occurredAt", "visitId", "patientRef", "type", "payload"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("event envelope missing %q: %v", key, first)
		}
	}
	if first["eventId"] != "EVT-000001" || first["type"] != "visit.opened" {
		t.Fatalf("first event = %v, want EVT-000001 visit.opened", first)
	}

	// Seed history is 1 opened + 1 placed = 2 events.
	_, tail := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000002", "")
	if got := len(tail["events"].([]any)); got != 0 {
		t.Fatalf("events after EVT-000002 = %d, want 0", got)
	}
	if tail["nextAfter"] != "" {
		t.Fatalf("nextAfter on empty page = %v, want empty", tail["nextAfter"])
	}
}

func TestListDemoVisits(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"visitType":"WALKIN","clinics":[{"clinicCode":"SURG"}]}`)

	status, visits := doList(t, app, http.MethodGet, "/api/v1/demo/visits", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(visits) != 2 {
		t.Fatalf("visits = %d, want seed + created", len(visits))
	}
	first := visits[0].(map[string]any)
	if first["visitId"] != "VISIT-001" {
		t.Fatalf("first visit = %v, want seeded VISIT-001", first["visitId"])
	}
}

// The full ADR-0009 example flow: pre-visit lab, clinic round, a mid-visit
// order inferring a return, encounter completed, cashier, done.
func TestDemoActionsSurfaceAsCanonicalEvents(t *testing.T) {
	app := New()
	_, xray := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"orderType":"XRAY","orderName":"Chest X-Ray","orderedByClinic":"MED"}`)
	ref := xray["orderRef"].(string)
	do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/performed", "")
	do(t, app, http.MethodPost, "/api/v1/demo/orders/"+ref+"/resulted", "")
	do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/clinics/MED/complete-encounter", "")
	do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/complete", "")

	_, feed := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000002&limit=100", "")
	want := []string{
		"order.placed",        // xray ordered
		"order.performed",     // xray performed
		"order.resulted",      // xray resulted
		"encounter.completed", // MED confirms done
		"visit.closed",        // visit completed
	}
	got := eventTypes(feed)
	if len(got) != len(want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// #22 follow-up: the console detail panel shows a QR of the visit id.
func TestVisitQrcode(t *testing.T) {
	app := New()

	status, png := doRaw(t, app, http.MethodGet, "/api/v1/demo/visits/VISIT-001/qrcode.png", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("body does not start with the PNG magic bytes")
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/demo/visits/NOPE/qrcode.png", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unknown visit qr: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown visit qr status = %d, want 404", resp.StatusCode)
	}
}

// Visit cancellation from the console: open orders are cancelled, the visit
// becomes CANCELLED, everything surfaces as canonical events.
func TestCancelVisit(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/cancel", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	if body["status"] != "CANCELLED" {
		t.Fatalf("visit status = %v, want CANCELLED", body["status"])
	}
	orders := body["orders"].([]any)
	if orders[0].(map[string]any)["status"] != "CANCELLED" {
		t.Fatalf("orders = %v, want the open lab order cancelled", orders)
	}

	_, feed := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000002&limit=100", "")
	want := []string{"order.cancelled", "visit.closed"}
	got := eventTypes(feed)
	if len(got) != len(want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}

	// Cancelling again is a no-op that adds no events.
	if status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/cancel", ""); status != http.StatusOK || body["status"] != "CANCELLED" {
		t.Fatalf("re-cancel = %d %v, want 200 no-op", status, body["status"])
	}
	_, feed = do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000004&limit=100", "")
	if n := len(feed["events"].([]any)); n != 0 {
		t.Fatalf("events after re-cancel = %d, want 0", n)
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/cancel", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit cancel = %d, want 404", status)
	}
}

func TestCancelCompletedVisitConflicts(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/complete", "")

	if status, resp := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/cancel", ""); status != http.StatusConflict {
		t.Fatalf("cancel COMPLETED visit = %d, want 409 (body %v)", status, resp)
	}
}

func TestConsoleServed(t *testing.T) {
	app := New()

	req, _ := http.NewRequest(http.MethodGet, "/console", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /console: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/console status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("/console content-type = %q, want text/html", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "Mock HIS Console") {
		t.Fatalf("/console body does not look like the console page")
	}
	if !strings.Contains(string(raw), "ORDER_TYPES") {
		t.Fatal("/console body is missing the ORDER_TYPES the pickers use")
	}

	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/console" {
		t.Fatalf("GET / = %d %q, want 302 to /console", resp.StatusCode, resp.Header.Get("Location"))
	}
}
