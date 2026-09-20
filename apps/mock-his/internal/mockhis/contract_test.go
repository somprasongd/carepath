package mockhis

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// The contract (packages/contracts/openapi/mock-his.yaml) is the source of
// truth for this boundary (ADR-0006, ADR-0008 amended by ADR-0009). This test
// pins its shape so a contract edit and the handler implementation cannot
// drift silently apart.

const contractPath = "../../../../packages/contracts/openapi/mock-his.yaml"

type property struct {
	Enum []string `yaml:"enum"`
}

type schema struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required"`
	Enum       []string            `yaml:"enum"`
	Properties map[string]property `yaml:"properties"`
}

type contract struct {
	Info struct {
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components struct {
		Schemas map[string]schema `yaml:"schemas"`
	} `yaml:"components"`
}

func loadContract(t *testing.T) contract {
	t.Helper()
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var c contract
	if err := yaml.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	return c
}

func has(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestContractDefinesCanonicalModel(t *testing.T) {
	c := loadContract(t)

	if c.Info.Version != "0.4.0" {
		t.Fatalf("contract version = %q, want 0.4.0", c.Info.Version)
	}

	// The two canonical read surfaces exist (ADR-0009): snapshot read, event
	// feed. There is no command endpoint — CarePath never writes to the HIS.
	if _, ok := c.Paths["/api/v1/visits/{visitId}"]["get"]; !ok {
		t.Fatal("contract is missing GET /api/v1/visits/{visitId}")
	}
	if _, ok := c.Paths["/api/v1/events"]["get"]; !ok {
		t.Fatal("contract is missing GET /api/v1/events")
	}
	if _, ok := c.Paths["/api/v1/visits/{visitId}/steps/{sequence}/transition"]; ok {
		t.Fatal("the step-transition command endpoint must not exist (ADR-0009 — the HIS never receives step commands)")
	}

	// Canonical enums, external ids, and the new fact vocabulary are spelled
	// out.
	if got := c.Components.Schemas["OrderStatus"].Enum; len(got) != 4 ||
		!has(got, "PLACED") || !has(got, "RESULTED") {
		t.Fatalf("OrderStatus enum = %v, want the 4 canonical statuses", got)
	}
	if got := c.Components.Schemas["EventType"].Enum; len(got) != 9 ||
		!has(got, "visit.opened") || !has(got, "encounter.started") || !has(got, "encounter.completed") || !has(got, "order.resulted") {
		t.Fatalf("EventType enum = %v, want the 9 canonical event types", got)
	}
	if got := c.Components.Schemas["Visit"].Required; !has(got, "visitId") || !has(got, "patientRef") {
		t.Fatalf("Visit required = %v, want external ids visitId and patientRef", got)
	}
	event := c.Components.Schemas["HISEvent"]
	for _, key := range []string{"eventId", "occurredAt", "visitId", "patientRef", "type", "payload"} {
		if !has(event.Required, key) {
			t.Fatalf("HISEvent required = %v, missing %q", event.Required, key)
		}
	}
	order := c.Components.Schemas["Order"]
	for _, key := range []string{"orderRef", "orderType", "orderName", "orderedByClinic", "orderedAt", "status"} {
		if !has(order.Required, key) {
			t.Fatalf("Order required = %v, missing %q", order.Required, key)
		}
	}

	// The contract carries only external facts; mapping to a service point
	// is CarePath-side configuration, so no CarePath id may appear on a
	// Visit/Order/Clinic.
	for _, forbidden := range []string{"servicePointId", "placeId", "stepKey"} {
		if _, ok := order.Properties[forbidden]; ok {
			t.Fatalf("Order must not carry %q (CarePath-side concern)", forbidden)
		}
		if _, ok := c.Components.Schemas["Visit"].Properties[forbidden]; ok {
			t.Fatalf("Visit must not carry %q (CarePath-side concern)", forbidden)
		}
	}
}
