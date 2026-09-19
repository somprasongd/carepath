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
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: decode response %q: %v", method, path, raw, err)
	}
	return resp.StatusCode, out
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
