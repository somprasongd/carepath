package mockhis

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// The contract (packages/contracts/openapi/mock-his.yaml) is the source of
// truth for this boundary (ADR-0006/ADR-0008). This test pins its shape so a
// contract edit and the handler implementation cannot drift silently apart.

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

type operation struct {
	RequestBody struct {
		Content map[string]struct {
			Schema struct {
				Ref string `yaml:"$ref"`
			} `yaml:"schema"`
		} `yaml:"content"`
	} `yaml:"requestBody"`
}

type contract struct {
	Info struct {
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths      map[string]map[string]operation `yaml:"paths"`
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

	if c.Info.Version != "0.2.0" {
		t.Fatalf("contract version = %q, want 0.2.0", c.Info.Version)
	}

	// The three integration surfaces exist (ADR-0008): snapshot read,
	// transition command, event feed.
	if _, ok := c.Paths["/api/v1/visits/{visitId}"]["get"]; !ok {
		t.Fatal("contract is missing GET /api/v1/visits/{visitId}")
	}
	transition, ok := c.Paths["/api/v1/visits/{visitId}/steps/{sequence}/transition"]["post"]
	if !ok {
		t.Fatal("contract is missing the transition command endpoint")
	}
	if _, ok := c.Paths["/api/v1/events"]["get"]; !ok {
		t.Fatal("contract is missing GET /api/v1/events")
	}
	if ref := transition.RequestBody.Content["application/json"].Schema.Ref; ref == "" {
		t.Fatal("transition endpoint has no request body schema")
	}

	// AC2: canonical enums, external ids, and idempotency keys are spelled out.
	if got := c.Components.Schemas["StepStatus"].Enum; len(got) != 5 ||
		!has(got, "STARTED") || !has(got, "CANCELLED") {
		t.Fatalf("StepStatus enum = %v, want the 5 canonical statuses", got)
	}
	if got := c.Components.Schemas["EventType"].Enum; len(got) != 6 ||
		!has(got, "visit.opened") || !has(got, "service.cancelled") {
		t.Fatalf("EventType enum = %v, want the 6 canonical event types", got)
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
	cmd := c.Components.Schemas["TransitionCommand"]
	if !has(cmd.Required, "commandId") || !has(cmd.Required, "to") {
		t.Fatalf("TransitionCommand required = %v, want commandId and to", cmd.Required)
	}
	if got := cmd.Properties["to"].Enum; len(got) != 3 || has(got, "READY") || has(got, "PENDING") {
		t.Fatalf("TransitionCommand.to enum = %v, want only STARTED/COMPLETED/CANCELLED", got)
	}

	// AC3: the contract carries only the external serviceCode; mapping to a
	// service point is CarePath-side configuration, so no CarePath id may
	// appear on a step.
	step := c.Components.Schemas["VisitStep"]
	if !has(step.Required, "serviceCode") {
		t.Fatalf("VisitStep required = %v, missing serviceCode", step.Required)
	}
	for _, forbidden := range []string{"servicePointId", "placeId"} {
		if _, ok := step.Properties[forbidden]; ok {
			t.Fatalf("VisitStep must not carry %q (CarePath-side concern)", forbidden)
		}
	}
}
