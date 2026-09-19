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
			name:   "decodes visit",
			status: http.StatusOK,
			body: `{"visitId":"VISIT-001","patientRef":"PAT-001","patientName":"สมชาย",` +
				`"visitType":"APPOINTMENT","status":"ACTIVE",` +
				`"clinics":[{"code":"MED"}],` +
				`"orders":[{"orderRef":"ORD-1","orderType":"LAB","orderName":"CBC",` +
				`"orderedByClinic":"MED","orderedAt":"2026-09-19T08:00:00+07:00","status":"PLACED"}],` +
				`"openedAt":"2026-09-19T09:00:00+07:00"}`,
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
			if !tt.wantVisit || visit.VisitID != "VISIT-001" || len(visit.Clinics) != 1 || len(visit.Orders) != 1 {
				t.Fatalf("visit = %+v, want decoded VISIT-001 with 1 clinic and 1 order", visit)
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
				Type:    his.EventOrderPerformed,
				Payload: map[string]any{"orderRef": "ORD-1"},
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
	if page.Events[0].Payload["orderRef"] != "ORD-1" {
		t.Fatalf("payload = %+v, want decoded orderRef ORD-1", page.Events[0].Payload)
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
