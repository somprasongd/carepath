// Planning-rules endpoint (#100): the surface is staff-read-only and needs
// neither the database nor the HIS, so the e2e here is about the guard and
// the wire shape — no token 401, EXECUTIVE 403, STAFF 200 with the phase
// ladder the planner actually runs on. The derivation itself (phases equal
// the planner's constants, examples equal Plan's output) is pinned in
// internal/journey/rules_test.go.
package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"carepath/apps/api/internal/platform/db"
)

func TestPlanningRulesGuardAndShape(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	app := newAnalyticsApp(t, database)

	// Unauthenticated is 401, not 403.
	resp, _ := get(t, app, "/api/v1/staff/planning-rules", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status %d, want 401", resp.StatusCode)
	}

	// EXECUTIVE reaches analytics, not the staff surface.
	exec := login(t, app, "exec", "demo")
	resp, _ = get(t, app, "/api/v1/staff/planning-rules", exec)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("exec: status %d, want 403", resp.StatusCode)
	}

	// STAFF gets the live-derived rules.
	staff := login(t, app, "staff", "demo")
	resp, body := get(t, app, "/api/v1/staff/planning-rules", staff)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff: status %d: %s, want 200", resp.StatusCode, body)
	}
	var rules struct {
		Phases []struct {
			Number int    `json:"number"`
			Key    string `json:"key"`
		} `json:"phases"`
		OrderTypes []struct {
			OrderType string `json:"orderType"`
			StepKind  string `json:"stepKind"`
		} `json:"orderTypes"`
		Examples []struct {
			ID    string `json:"id"`
			Steps []struct {
				StepKey string `json:"stepKey"`
				Status  string `json:"status"`
			} `json:"steps"`
		} `json:"examples"`
	}
	if err := json.Unmarshal([]byte(body), &rules); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	if len(rules.Phases) != 7 || rules.Phases[0].Key != "REGISTRATION" || rules.Phases[6].Key != "PHARMACY" {
		t.Fatalf("phases = %+v, want the planner's 7-phase ladder", rules.Phases)
	}
	if len(rules.OrderTypes) != 5 {
		t.Fatalf("orderTypes = %+v, want the 5 canonical HIS order types", rules.OrderTypes)
	}
	if len(rules.Examples) == 0 || len(rules.Examples[0].Steps) == 0 {
		t.Fatalf("examples empty: %s", body)
	}
}
