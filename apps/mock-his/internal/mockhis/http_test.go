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

func stepStatuses(body map[string]any) map[int]string {
	t := map[int]string{}
	for _, raw := range body["steps"].([]any) {
		step := raw.(map[string]any)
		t[int(step["sequence"].(float64))] = step["status"].(string)
	}
	return t
}

const cmdStarted = `{"commandId":"11111111-1111-1111-1111-111111111111","to":"STARTED"}`
const cmdStarted2 = `{"commandId":"22222222-2222-2222-2222-222222222222","to":"STARTED"}`
const cmdCompleted = `{"commandId":"33333333-3333-3333-3333-333333333333","to":"COMPLETED"}`

func TestVisitSnapshotMatchesContract(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, body)
	}
	if body["visitId"] != "VISIT-001" || body["patientRef"] != "PATIENT-DEMO-001" || body["status"] != "ACTIVE" {
		t.Fatalf("snapshot header = %v, want VISIT-001 / PATIENT-DEMO-001 / ACTIVE", body)
	}
	want := map[int]string{1: "COMPLETED", 2: "COMPLETED", 3: "COMPLETED", 4: "READY", 5: "PENDING"}
	got := stepStatuses(body)
	if len(got) != len(want) {
		t.Fatalf("steps = %v, want %v", got, want)
	}
	for seq, st := range want {
		if got[seq] != st {
			t.Fatalf("step %d status = %q, want %q", seq, got[seq], st)
		}
	}

	if status, _ := do(t, app, http.MethodGet, "/api/v1/visits/NOPE", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}
}

func TestTransitionCommandIsIdempotent(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodPost,
		"/api/v1/visits/VISIT-001/steps/4/transition", cmdStarted)
	if status != http.StatusOK || body["status"] != "STARTED" {
		t.Fatalf("first transition = %d %v, want 200 STARTED", status, body)
	}

	// Same commandId replayed: no-op success, no second service.started event.
	status, body = do(t, app, http.MethodPost,
		"/api/v1/visits/VISIT-001/steps/4/transition", cmdStarted)
	if status != http.StatusOK || body["status"] != "STARTED" {
		t.Fatalf("replayed commandId = %d %v, want 200 STARTED", status, body)
	}
	_, feed := do(t, app, http.MethodGet, "/api/v1/events?limit=200", "")
	started := 0
	for _, raw := range feed["events"].([]any) {
		if raw.(map[string]any)["type"] == "service.started" {
			started++
		}
	}
	if started != 1 {
		t.Fatalf("service.started events = %d, want 1 (replay must not duplicate)", started)
	}

	// Different commandId targeting the current status: also a no-op success.
	status, body = do(t, app, http.MethodPost,
		"/api/v1/visits/VISIT-001/steps/4/transition", cmdStarted2)
	if status != http.StatusOK || body["status"] != "STARTED" {
		t.Fatalf("same-state transition = %d %v, want 200 STARTED", status, body)
	}
}

func TestTransitionValidationAndConflicts(t *testing.T) {
	app := New()
	transition := func(path, body string) int {
		status, resp := do(t, app, http.MethodPost, path, body)
		if _, ok := resp["error"]; status >= 400 && !ok {
			t.Fatalf("error response %v has no \"error\" field", resp)
		}
		return status
	}

	if s := transition("/api/v1/visits/VISIT-001/steps/4/transition", `{"commandId":"c1","to":"READY"}`); s != http.StatusBadRequest {
		t.Fatalf("non-command target READY = %d, want 400", s)
	}
	if s := transition("/api/v1/visits/VISIT-001/steps/4/transition", `{"to":"STARTED"}`); s != http.StatusBadRequest {
		t.Fatalf("missing commandId = %d, want 400", s)
	}
	if s := transition("/api/v1/visits/VISIT-001/steps/4/transition", `not json`); s != http.StatusBadRequest {
		t.Fatalf("invalid body = %d, want 400", s)
	}
	if s := transition("/api/v1/visits/NOPE/steps/4/transition", cmdStarted); s != http.StatusNotFound {
		t.Fatalf("unknown visit = %d, want 404", s)
	}
	if s := transition("/api/v1/visits/VISIT-001/steps/99/transition", cmdStarted); s != http.StatusNotFound {
		t.Fatalf("unknown step = %d, want 404", s)
	}
	// Terminal step: new transitions conflict, same-status is a no-op.
	if s := transition("/api/v1/visits/VISIT-001/steps/1/transition", cmdStarted); s != http.StatusConflict {
		t.Fatalf("STARTED on COMPLETED step = %d, want 409", s)
	}
	if s := transition("/api/v1/visits/VISIT-001/steps/1/transition", cmdCompleted); s != http.StatusOK {
		t.Fatalf("COMPLETED on COMPLETED step = %d, want 200 no-op", s)
	}
}

func TestCompleteStepReadiesNextAndCompletesVisit(t *testing.T) {
	app := New()
	post := func(path, commandID, to string) int {
		status, _ := do(t, app, http.MethodPost, path,
			`{"commandId":"`+commandID+`","to":"`+to+`"}`)
		return status
	}

	// LAB: started then completed — PHARMACY (the next PENDING) becomes READY.
	post("/api/v1/visits/VISIT-001/steps/4/transition", "44444444-4444-4444-4444-444444444444", "STARTED")
	post("/api/v1/visits/VISIT-001/steps/4/transition", "55555555-5555-5555-5555-555555555555", "COMPLETED")
	_, snap := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	got := stepStatuses(snap)
	if got[4] != "COMPLETED" || got[5] != "READY" || snap["status"] != "ACTIVE" {
		t.Fatalf("after LAB completed: steps = %v visit = %v, want PHARMACY READY and visit ACTIVE", got, snap["status"])
	}

	// PHARMACY: finishing the last open step completes the visit.
	post("/api/v1/visits/VISIT-001/steps/5/transition", "66666666-6666-6666-6666-666666666666", "STARTED")
	post("/api/v1/visits/VISIT-001/steps/5/transition", "77777777-7777-7777-7777-777777777777", "COMPLETED")
	_, snap = do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	if snap["status"] != "COMPLETED" {
		t.Fatalf("visit status = %v, want COMPLETED after last step", snap["status"])
	}

	// The feed records the full history: seed events plus the transitions.
	_, feed := do(t, app, http.MethodGet, "/api/v1/events?limit=200", "")
	var types []string
	for _, raw := range feed["events"].([]any) {
		types = append(types, raw.(map[string]any)["type"].(string))
	}
	want := []string{
		"visit.opened",
		"service.requested", "service.requested", "service.requested", "service.requested", "service.requested",
		"service.completed", "service.completed", "service.completed",
		"service.started",   // LAB
		"service.completed", // LAB
		"visit.updated",     // PHARMACY became READY
		"service.started",   // PHARMACY
		"service.completed", // PHARMACY
		"visit.updated",     // visit COMPLETED
	}
	if len(types) != len(want) {
		t.Fatalf("event count = %d (%v), want %d", len(types), types, len(want))
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, types[i], want[i])
		}
	}
}

func TestEventFeedCursorAndEnvelope(t *testing.T) {
	app := New()

	status, page := do(t, app, http.MethodGet, "/api/v1/events?limit=3", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	events := page["events"].([]any)
	if len(events) != 3 || page["nextAfter"] != "EVT-000003" {
		t.Fatalf("first page = %v nextAfter %v, want 3 events and EVT-000003", len(events), page["nextAfter"])
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

	_, page = do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000003&limit=2", "")
	events = page["events"].([]any)
	if len(events) != 2 || events[0].(map[string]any)["eventId"] != "EVT-000004" {
		t.Fatalf("second page starts at %v, want EVT-000004", events[0])
	}

	// Seed history is 1 opened + 5 requested + 3 completed = 9 events.
	_, tail := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000009", "")
	if got := len(tail["events"].([]any)); got != 0 {
		t.Fatalf("events after EVT-000009 = %d, want 0", got)
	}
	if tail["nextAfter"] != "" {
		t.Fatalf("nextAfter on empty page = %v, want empty", tail["nextAfter"])
	}
}

// #22 AC1: the console can create a demo visit (or pick the seeded one).
func TestCreateDemoVisit(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits",
		`{"patientRef":"PATIENT-X","serviceCodes":["REGISTRATION","XRAY"]}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %v)", status, body)
	}
	if body["visitId"] != "VISIT-002" || body["patientRef"] != "PATIENT-X" || body["status"] != "ACTIVE" {
		t.Fatalf("created visit = %v, want VISIT-002 / PATIENT-X / ACTIVE", body)
	}
	got := stepStatuses(body)
	if len(got) != 2 || got[1] != "READY" || got[2] != "PENDING" {
		t.Fatalf("created steps = %v, want first READY rest PENDING", got)
	}

	// Defaults: no patientRef and no codes -> standard template, ids assigned.
	status, body = do(t, app, http.MethodPost, "/api/v1/demo/visits", `{}`)
	if status != http.StatusCreated {
		t.Fatalf("default create status = %d, want 201", status)
	}
	if body["visitId"] != "VISIT-003" || body["patientRef"] != "PATIENT-DEMO-003" {
		t.Fatalf("defaults = %v, want VISIT-003 / PATIENT-DEMO-003", body)
	}
	if len(body["steps"].([]any)) != len(defaultServiceCodes) {
		t.Fatalf("default steps = %d, want the standard template", len(body["steps"].([]any)))
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits", `not json`); status != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"serviceCodes":["  "]}`); status != http.StatusBadRequest {
		t.Fatalf("blank service code status = %d, want 400", status)
	}
}

func TestListDemoVisits(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits", `{}`)

	status, visits := doList(t, app, http.MethodGet, "/api/v1/demo/visits", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(visits) != 2 {
		t.Fatalf("visits = %d, want seed + created", len(visits))
	}
	first := visits[0].(map[string]any)
	if first["visitId"] != "VISIT-001" || len(first["steps"].([]any)) != 5 {
		t.Fatalf("first visit = %v, want seeded VISIT-001 with 5 steps", first["visitId"])
	}
}

// #22 AC2: an order (e.g. X-Ray) can be sent to a visit.
func TestAddOrder(t *testing.T) {
	app := New()

	status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders",
		`{"serviceCode":"XRAY"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %v)", status, body)
	}
	if body["sequence"].(float64) != 6 || body["serviceCode"] != "XRAY" || body["status"] != "PENDING" {
		t.Fatalf("order step = %v, want sequence 6 XRAY PENDING", body)
	}

	// The appended order behaves like any step: completing LAB readies the
	// next PENDING (PHARMACY, not XRAY).
	post := func(path, commandID, to string) int {
		s, _ := do(t, app, http.MethodPost, path, `{"commandId":"`+commandID+`","to":"`+to+`"}`)
		return s
	}
	post("/api/v1/visits/VISIT-001/steps/4/transition", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "STARTED")
	post("/api/v1/visits/VISIT-001/steps/4/transition", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "COMPLETED")
	_, snap := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	got := stepStatuses(snap)
	if got[5] != "READY" || got[6] != "PENDING" {
		t.Fatalf("after LAB completed = %v, want PHARMACY(5) READY and XRAY(6) PENDING", got)
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/orders", `{"serviceCode":"XRAY"}`); status != http.StatusNotFound {
		t.Fatalf("unknown visit status = %d, want 404", status)
	}
	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/orders", `{"serviceCode":"  "}`); status != http.StatusBadRequest {
		t.Fatalf("blank code status = %d, want 400", status)
	}
}

func TestAddOrderConflictOnFinishedVisit(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"serviceCodes":["REGISTRATION"]}`)
	do(t, app, http.MethodPost, "/api/v1/visits/VISIT-002/steps/1/transition",
		`{"commandId":"cccccccc-cccc-cccc-cccc-cccccccccccc","to":"STARTED"}`)
	do(t, app, http.MethodPost, "/api/v1/visits/VISIT-002/steps/1/transition",
		`{"commandId":"dddddddd-dddd-dddd-dddd-dddddddddddd","to":"COMPLETED"}`)

	if status, resp := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-002/orders", `{"serviceCode":"XRAY"}`); status != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %v)", status, resp)
	}
}

// #22 AC4 at the HIS boundary: demo actions are announced as canonical
// events only — nothing here knows about CarePath or its database.
func TestDemoActionsSurfaceAsCanonicalEvents(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"serviceCodes":["REGISTRATION","LAB"]}`)
	do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-002/orders", `{"serviceCode":"XRAY"}`)
	do(t, app, http.MethodPost, "/api/v1/visits/VISIT-002/steps/1/transition",
		`{"commandId":"dddddddd-dddd-dddd-dddd-dddddddddddd","to":"COMPLETED"}`)

	_, feed := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000009&limit=100", "")
	var types []string
	for _, raw := range feed["events"].([]any) {
		types = append(types, raw.(map[string]any)["type"].(string))
	}
	want := []string{
		"visit.opened",      // demo create
		"service.requested", // REGISTRATION
		"service.requested", // LAB
		"service.requested", // XRAY order
		"service.completed", // REGISTRATION completed via console
		"visit.updated",     // LAB became READY
	}
	if len(types) != len(want) {
		t.Fatalf("event types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, types[i], want[i])
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

// Visit cancellation from the console: open steps are cancelled, the visit
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
	got := stepStatuses(body)
	if got[1] != "COMPLETED" || got[4] != "CANCELLED" || got[5] != "CANCELLED" {
		t.Fatalf("steps = %v, want completed kept and open steps cancelled", got)
	}

	_, feed := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000009&limit=100", "")
	var types []string
	for _, raw := range feed["events"].([]any) {
		types = append(types, raw.(map[string]any)["type"].(string))
	}
	want := []string{
		"service.cancelled", // LAB
		"service.cancelled", // PHARMACY
		"visit.updated",     // visit CANCELLED
	}
	if len(types) != len(want) {
		t.Fatalf("event types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, types[i], want[i])
		}
	}

	// Cancelling again is a no-op that adds no events.
	if status, body := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-001/cancel", ""); status != http.StatusOK || body["status"] != "CANCELLED" {
		t.Fatalf("re-cancel = %d %v, want 200 no-op", status, body["status"])
	}
	_, feed = do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000012&limit=100", "")
	if n := len(feed["events"].([]any)); n != 0 {
		t.Fatalf("events after re-cancel = %d, want 0", n)
	}

	if status, _ := do(t, app, http.MethodPost, "/api/v1/demo/visits/NOPE/cancel", ""); status != http.StatusNotFound {
		t.Fatalf("unknown visit cancel = %d, want 404", status)
	}
}

func TestCancelCompletedVisitConflicts(t *testing.T) {
	app := New()
	do(t, app, http.MethodPost, "/api/v1/demo/visits", `{"serviceCodes":["REGISTRATION"]}`)
	do(t, app, http.MethodPost, "/api/v1/visits/VISIT-002/steps/1/transition",
		`{"commandId":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee","to":"COMPLETED"}`)

	if status, resp := do(t, app, http.MethodPost, "/api/v1/demo/visits/VISIT-002/cancel", ""); status != http.StatusConflict {
		t.Fatalf("cancel COMPLETED visit = %d, want 409 (body %v)", status, resp)
	}
}

// Closing the last open step by cancelling it cancels the visit (the
// completion check used to run on completions only).
func TestCancellingLastOpenStepCancelsVisit(t *testing.T) {
	app := New()
	cancel := func(path, id string) int {
		s, _ := do(t, app, http.MethodPost, path, `{"commandId":"`+id+`","to":"CANCELLED"}`)
		return s
	}

	if s := cancel("/api/v1/visits/VISIT-001/steps/4/transition", "ffffffff-ffff-ffff-ffff-ffffffffffff"); s != http.StatusOK {
		t.Fatalf("cancel LAB = %d, want 200", s)
	}
	_, snap := do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	if snap["status"] != "ACTIVE" {
		t.Fatalf("visit = %v after one cancel, want still ACTIVE (PHARMACY open)", snap["status"])
	}

	if s := cancel("/api/v1/visits/VISIT-001/steps/5/transition", "abababab-abab-abab-abab-abababababab"); s != http.StatusOK {
		t.Fatalf("cancel PHARMACY = %d, want 200", s)
	}
	_, snap = do(t, app, http.MethodGet, "/api/v1/visits/VISIT-001", "")
	if snap["status"] != "CANCELLED" {
		t.Fatalf("visit = %v after cancelling the last open step, want CANCELLED", snap["status"])
	}
	got := stepStatuses(snap)
	if got[4] != "CANCELLED" || got[5] != "CANCELLED" || got[1] != "COMPLETED" {
		t.Fatalf("steps = %v, want completed kept and cancelled stays cancelled", got)
	}

	_, feed := do(t, app, http.MethodGet, "/api/v1/events?after=EVT-000009&limit=100", "")
	var last string
	for _, raw := range feed["events"].([]any) {
		last = raw.(map[string]any)["type"].(string)
	}
	if last != "visit.updated" {
		t.Fatalf("last event = %q, want visit.updated (visit cancelled)", last)
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
	// Service selection is dropdown-based (no free-text typos): the page must
	// carry the service catalog the pickers populate from.
	if !strings.Contains(string(raw), "SERVICE_CATALOG") {
		t.Fatal("/console body is missing the SERVICE_CATALOG the pickers use")
	}
	for _, code := range []string{"REGISTRATION", "SCREENING", "DOCTOR", "LAB", "PHARMACY", "XRAY", "CT"} {
		if !strings.Contains(string(raw), `"`+code+`"`) {
			t.Fatalf("/console catalog is missing %q", code)
		}
	}
	if strings.Contains(string(raw), `id="new-codes"`) {
		t.Fatal("/console still has the free-text new-codes input")
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
