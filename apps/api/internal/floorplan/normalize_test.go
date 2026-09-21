package floorplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"carepath/apps/api/internal/platform/apperr"
)

// shippedPlan reads one of the floor-plan assets the MVP ships with. They
// are the only real input this package has, so they are the baseline: a
// change that stops accepting them is a change that breaks the product.
func shippedPlan(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "packages", "floorplans", "floors", name)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return src
}

func TestNormalizeAcceptsShippedPlans(t *testing.T) {
	cases := []struct {
		file    string
		floorID string
	}{
		{"i-1301-ground.svg", "I-1301"},
		{"i-1302-upper.svg", "I-1302"},
	}

	for _, tc := range cases {
		t.Run(tc.floorID, func(t *testing.T) {
			plan, err := Normalize(shippedPlan(t, tc.file), Expect{FloorID: tc.floorID})
			if err != nil {
				t.Fatalf("Normalize() rejected the shipped plan: %v", err)
			}
			if plan.ViewBox != "0 0 1600 900" {
				t.Errorf("ViewBox = %q, want %q", plan.ViewBox, "0 0 1600 900")
			}
			if len(plan.SHA256) != 64 {
				t.Errorf("SHA256 = %q, want 64 hex characters", plan.SHA256)
			}

			// The conventions apps/web joins on must survive normalization.
			for _, want := range []string{
				`data-floor="` + tc.floorID + `"`,
				`id="route-layer"`,
				`data-place-id=`,
				`viewBox="0 0 1600 900"`,
			} {
				if !strings.Contains(plan.SVG, want) {
					t.Errorf("normalized plan lost %s", want)
				}
			}
		})
	}
}

// The rewrite that lets a plan keep its own colours without restyling the
// app around it. :root would match the document root once inlined — and
// match nothing at all inside a shadow root.
func TestNormalizeScopesRootSelector(t *testing.T) {
	plan, err := Normalize(shippedPlan(t, "i-1301-ground.svg"), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if strings.Contains(plan.SVG, ":root") {
		t.Error("normalized plan still declares :root — it would restyle the whole app")
	}
	if !strings.Contains(plan.SVG, "svg{") {
		t.Error("the :root block did not survive as an svg block — the plan would lose its colours")
	}
	if !strings.Contains(plan.SVG, "--wall:") {
		t.Error("normalized plan lost its custom properties")
	}
}

// Normalizing twice must change nothing. If it did, the serializer would be
// emitting something the parser does not accept, which is exactly the class
// of mismatch that lets markup through a sanitizer.
func TestNormalizeIsIdempotent(t *testing.T) {
	first, err := Normalize(shippedPlan(t, "i-1301-ground.svg"), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("first Normalize() error: %v", err)
	}
	second, err := Normalize([]byte(first.SVG), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("re-normalizing our own output failed: %v", err)
	}
	if first.SHA256 != second.SHA256 {
		t.Errorf("digest changed on re-normalize: %s then %s", first.SHA256, second.SHA256)
	}
}

// Attribute order is not meaning. Two spellings of the same drawing must
// land on one stored plan, not two.
func TestNormalizeDigestIgnoresAttributeOrder(t *testing.T) {
	a := planWith(`<rect data-place-id="REG-01" class="room" x="1" y="2"/>`)
	b := planWith(`<rect x="1" class="room" y="2" data-place-id="REG-01"/>`)

	planA, err := Normalize([]byte(a), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("Normalize(a) error: %v", err)
	}
	planB, err := Normalize([]byte(b), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("Normalize(b) error: %v", err)
	}
	if planA.SHA256 != planB.SHA256 {
		t.Errorf("same drawing hashed differently: %s vs %s", planA.SHA256, planB.SHA256)
	}
}

func TestNormalizeReportsModelMismatches(t *testing.T) {
	src := planWith(`<rect data-place-id="REG-01"/><circle id="node-reception"/><rect data-place-id="GHOST-01"/>`)

	plan, err := Normalize([]byte(src), Expect{
		FloorID:  "I-1301",
		PlaceIDs: []string{"REG-01", "LAB-01"},
		NodeIDs:  []string{"node-reception", "node-lab"},
	})
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	want := []Warning{
		{Code: WarnNodeMissing, Ref: "node-lab"},
		{Code: WarnPlaceMissing, Ref: "LAB-01"},
		{Code: WarnPlaceUnknown, Ref: "GHOST-01"},
	}
	if len(plan.Warnings) != len(want) {
		t.Fatalf("Warnings = %+v, want %+v", plan.Warnings, want)
	}
	for i, w := range want {
		if plan.Warnings[i] != w {
			t.Errorf("Warnings[%d] = %+v, want %+v", i, plan.Warnings[i], w)
		}
	}
}

// A mismatched viewBox is the failure with no symptom: nothing errors, every
// pin and route just lands somewhere else. It has to be a rejection.
func TestNormalizeHoldsTheCoordinateSpace(t *testing.T) {
	src := strings.Replace(planWith(""), `viewBox="0 0 1600 900"`, `viewBox="0 0 800 450"`, 1)

	if _, err := Normalize([]byte(src), Expect{FloorID: "I-1301", ViewBox: "0 0 1600 900"}); err == nil {
		t.Fatal("Normalize() accepted a plan drawn in a different coordinate space")
	} else if apperr.KindOf(err) != apperr.KindInvalid {
		t.Errorf("KindOf(err) = %v, want KindInvalid", apperr.KindOf(err))
	}

	// The first plan for a floor defines the space instead of being held to one.
	plan, err := Normalize([]byte(src), Expect{FloorID: "I-1301"})
	if err != nil {
		t.Fatalf("first upload rejected: %v", err)
	}
	if plan.ViewBox != "0 0 800 450" {
		t.Errorf("ViewBox = %q, want %q", plan.ViewBox, "0 0 800 450")
	}
}

func TestNormalizeChecksFloorIdentity(t *testing.T) {
	cases := map[string]string{
		"wrong floor":  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900"><g data-floor="I-1302"><g id="route-layer"/></g></svg>`,
		"no floor":     `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900"><g id="route-layer"/></svg>`,
		"two floors":   `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900"><g data-floor="I-1301"/><g data-floor="I-1302"><g id="route-layer"/></g></svg>`,
		"no route ref": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900"><g data-floor="I-1301"/></svg>`,
		// FloorPlanMap only ever queries g[data-floor] to find the group it
		// measures to fit the plan into the map card — a marker on any other
		// element is invisible to that query, so it must be rejected here
		// rather than silently accepted and rendered unfitted.
		"floor marker not on a <g>": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900">` +
			`<rect data-floor="I-1301"/><g id="route-layer"/></svg>`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Normalize([]byte(src), Expect{FloorID: "I-1301"}); err == nil {
				t.Error("Normalize() accepted it")
			}
		})
	}
}

func TestNormalizeEnforcesLimits(t *testing.T) {
	t.Run("file size", func(t *testing.T) {
		if _, err := Normalize(make([]byte, MaxBytes+1), Expect{FloorID: "I-1301"}); err != ErrTooLarge {
			t.Errorf("err = %v, want ErrTooLarge", err)
		}
	})

	t.Run("element count", func(t *testing.T) {
		src := planWith(strings.Repeat(`<rect/>`, MaxElements+1))
		if _, err := Normalize([]byte(src), Expect{FloorID: "I-1301"}); err == nil {
			t.Error("Normalize() accepted a plan over the element cap")
		}
	})

	t.Run("css size", func(t *testing.T) {
		src := planWith(`<style>` + strings.Repeat(".a{fill:#fff}", MaxStyleBytes/13+1) + `</style>`)
		if _, err := Normalize([]byte(src), Expect{FloorID: "I-1301"}); err == nil {
			t.Error("Normalize() accepted a plan over the CSS cap")
		}
	})
}

// checkURLRefs used to find "url(" in a lowercased copy of the value and
// then slice the ORIGINAL string by that index. strings.ToLower is not
// byte-length preserving for every rune (e.g. "İ" is 2 bytes, "i̇" is 3), so
// a legitimate non-ASCII label ahead of a same-file url(#...) reference
// could desync the index and read a "url(" byte sequence that was never
// there — see indexFoldASCII in normalize.go.
func TestNormalizeAcceptsSameFileURLRefsAfterNonASCIIText(t *testing.T) {
	// U+0130 LATIN CAPITAL LETTER I WITH DOT ABOVE, ahead of a legitimate
	// same-file marker reference — the case that used to desync the index.
	src := planWith(`<text>İİİİİİİİİİİİİİİİİİİİİİ</text>` +
		`<marker id="arrow"/>` +
		`<polyline points="0,0 1,1" marker-end="url(#arrow)"/>`)
	if _, err := Normalize([]byte(src), Expect{FloorID: "I-1301"}); err != nil {
		t.Errorf("Normalize() rejected a same-file url() ref after non-ASCII text: %v", err)
	}
}

// planWith wraps a fragment in the smallest plan that satisfies the
// structural rules, so a case only has to express the thing under test.
func planWith(inner string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900">` +
		`<g data-floor="I-1301"><g id="route-layer"/>` + inner + `</g></svg>`
}
