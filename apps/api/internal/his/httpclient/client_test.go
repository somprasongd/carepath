package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
)

func TestGetVisit(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantVisit  bool
		wantKind   apperr.Kind
		wantErrors bool
	}{
		{
			name:      "decodes visit",
			status:    http.StatusOK,
			body:      `{"visitId":"VISIT-001","patientRef":"PAT-001","status":"ACTIVE","steps":[{"sequence":1,"serviceCode":"LAB","status":"READY"}]}`,
			wantVisit: true,
		},
		{name: "not found", status: http.StatusNotFound, body: `{"error":"visit not found"}`, wantKind: apperr.KindNotFound, wantErrors: true},
		{name: "upstream error", status: http.StatusInternalServerError, body: `{"error":"boom"}`, wantKind: apperr.KindUpstream, wantErrors: true},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantKind: apperr.KindUpstream, wantErrors: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			visit, err := New(server.URL, server.Client()).GetVisit(context.Background(), "VISIT-001")
			if tt.wantErrors {
				if err == nil || apperr.KindOf(err) != tt.wantKind {
					t.Fatalf("error = %v, want kind %v", err, tt.wantKind)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetVisit: %v", err)
			}
			if !tt.wantVisit || visit.VisitID != "VISIT-001" || len(visit.Steps) != 1 {
				t.Fatalf("visit = %+v, want decoded VISIT-001 with 1 step", visit)
			}
		})
	}
}

func TestEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("after"); got != "EVT-000002" {
			t.Errorf("after = %q, want EVT-000002", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(his.EventPage{
			Events: []his.Event{{
				EventID: "EVT-000003", VisitID: "VISIT-001", PatientRef: "PAT-001",
				Type:    his.EventServiceCompleted,
				Payload: map[string]any{"sequence": float64(4), "serviceCode": "LAB"},
			}},
			NextAfter: "EVT-000003",
		})
	}))
	defer server.Close()

	page, err := New(server.URL, server.Client()).Events(context.Background(), "EVT-000002", 100)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(page.Events) != 1 || page.Events[0].EventID != "EVT-000003" || page.NextAfter != "EVT-000003" {
		t.Fatalf("page = %+v, want one event and cursor EVT-000003", page)
	}
	if page.Events[0].Payload["serviceCode"] != "LAB" {
		t.Fatalf("payload = %+v, want decoded serviceCode LAB", page.Events[0].Payload)
	}
}

func TestEventsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := New(server.URL, server.Client()).Events(context.Background(), "", 0)
	if err == nil || apperr.KindOf(err) != apperr.KindUpstream {
		t.Fatalf("error = %v, want KindUpstream", err)
	}
}

func TestTransitionStep(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantStep bool
		wantKind apperr.Kind
		wantMsg  string
	}{
		{
			name:     "decodes resulting step",
			status:   http.StatusOK,
			body:     `{"sequence":2,"serviceCode":"LAB","status":"STARTED"}`,
			wantStep: true,
		},
		{name: "unknown target", status: http.StatusBadRequest, body: `{"error":"unknown target status \"PAUSED\""}`, wantKind: apperr.KindInvalid, wantMsg: `unknown target status "PAUSED"`},
		{name: "visit not found", status: http.StatusNotFound, body: `{"error":"visit not found"}`, wantKind: apperr.KindNotFound, wantMsg: "visit not found"},
		{name: "illegal transition", status: http.StatusConflict, body: `{"error":"step 1 is COMPLETED and cannot transition to STARTED"}`, wantKind: apperr.KindConflict, wantMsg: "step 1 is COMPLETED and cannot transition to STARTED"},
		{name: "upstream error", status: http.StatusInternalServerError, body: `{"error":"boom"}`, wantKind: apperr.KindUpstream},
		{name: "error without message", status: http.StatusConflict, body: `{}`, wantKind: apperr.KindConflict, wantMsg: "HIS rejected the command"},
		{name: "malformed success body", status: http.StatusOK, body: `{`, wantKind: apperr.KindUpstream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/visits/VISIT-001/steps/2/transition" {
					t.Errorf("request = %s %s, want POST /api/v1/visits/VISIT-001/steps/2/transition", r.Method, r.URL.Path)
				}
				var cmd his.TransitionCommand
				if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil || cmd.CommandID != "CMD-1" || cmd.To != "STARTED" {
					t.Errorf("command body = %+v (err %v), want commandId CMD-1 to STARTED", cmd, err)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			step, err := New(server.URL, server.Client()).TransitionStep(context.Background(), "VISIT-001", 2,
				his.TransitionCommand{CommandID: "CMD-1", To: "STARTED"})
			if tt.wantStep {
				if err != nil {
					t.Fatalf("TransitionStep: %v", err)
				}
				if step.Sequence != 2 || step.ServiceCode != "LAB" || step.Status != "STARTED" {
					t.Fatalf("step = %+v, want decoded LAB step", step)
				}
				return
			}
			if err == nil || apperr.KindOf(err) != tt.wantKind {
				t.Fatalf("error = %v, want kind %v", err, tt.wantKind)
			}
			if tt.wantMsg != "" && err.Error() != tt.wantMsg {
				t.Fatalf("message = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
