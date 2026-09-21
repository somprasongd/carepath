package floorplan

import (
	"strings"
	"testing"

	"carepath/apps/api/internal/platform/apperr"
)

// The corpus below is the reason this package exists. apps/web inlines the
// plan into the patient app's DOM (FloorPlanMap injects it so the route and
// the destination pin can be drawn into real elements), and the patient app
// holds a session token. Every case here is something an uploaded file could
// otherwise do to that page.
//
// Cases are grouped by what they attack, and each one states the mechanism
// rather than just the payload — a future change that "fixes" a test by
// loosening the allowlist should have to argue with the comment.
var rejected = []struct {
	name string
	svg  string
}{
	// --- script execution ---------------------------------------------
	{
		"script element",
		planWith(`<script>alert(1)</script>`),
	},
	{
		"event handler attribute",
		planWith(`<g onload="alert(1)"><rect/></g>`),
	},
	{
		"event handler on a shape",
		planWith(`<rect onclick="alert(1)" width="10" height="10"/>`),
	},
	{
		// SVG's HTML escape hatch: everything inside is parsed as HTML.
		"foreignObject carrying HTML",
		planWith(`<foreignObject><div>x</div></foreignObject>`),
	},
	{
		// SMIL can set an attribute after the allowlist has already run.
		"animation element",
		planWith(`<animate attributeName="fill" to="red"/>`),
	},

	// --- namespace confusion ------------------------------------------
	{
		// Same local name, different namespace: an xhtml <svg> is not a
		// drawing, and everything below it would be HTML.
		"root in the xhtml namespace",
		`<svg xmlns="http://www.w3.org/1999/xhtml" viewBox="0 0 1600 900"><g data-floor="I-1301"><g id="route-layer"/></g></svg>`,
	},
	{
		"xlink href attribute",
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 1600 900">` +
			`<g data-floor="I-1301"><g id="route-layer"/><text xlink:href="javascript:alert(1)">x</text></g></svg>`,
	},

	// --- external references ------------------------------------------
	{
		"image pointing off-site",
		planWith(`<image href="https://evil.example/pixel.png"/>`),
	},
	{
		"use pulling in a remote fragment",
		planWith(`<use href="https://evil.example/x.svg#a"/>`),
	},
	{
		"paint server pointing off-site",
		planWith(`<rect fill="url(https://evil.example/x)" width="1" height="1"/>`),
	},
	{
		"marker pointing off-site",
		planWith(`<polyline points="0,0 1,1" marker-end="url(https://evil.example/x#a)"/>`),
	},

	// --- CSS ----------------------------------------------------------
	{
		"css import",
		planWith(`<style>@import url(#a);.a{fill:#fff}</style>`),
	},
	{
		"css fetching a remote background",
		planWith(`<style>.a{background:url(https://evil.example/x.png)}</style>`),
	},
	{
		// The classic raw-text breakout. The XML parser sees the </style>
		// as a close tag and <script> as a new element; wrapped in CDATA it
		// stays character data instead, which is why CSS may not contain
		// '<' at all.
		"style body escaping itself through CDATA",
		planWith(`<style><![CDATA[.a{} </style><script>alert(1)</script>]]></style>`),
	},
	{
		// Same breakout written as character references, for a parser that
		// decodes them on the way back out.
		"style body escaping itself through entities",
		planWith(`<style>.a{} &lt;/style&gt;&lt;script&gt;alert(1)&lt;/script&gt;</style>`),
	},
	{
		"style attribute fetching a remote resource",
		planWith(`<rect style="background:url(https://evil.example/x)" width="1" height="1"/>`),
	},
	{
		"shadow-piercing selector",
		planWith(`<style>:host{display:none}</style>`),
	},

	// --- javascript: urls, including obfuscated ------------------------
	{
		"javascript url in a style attribute",
		planWith(`<text style="color:red;background:javascript:alert(1)">x</text>`),
	},
	{
		// The XML parser decodes &#9; to a tab before we ever see the
		// value, so the check runs on what the browser would act on.
		"javascript url split by a character reference",
		planWith(`<text style="background:java&#9;script:alert(1)">x</text>`),
	},

	// --- parser-level -------------------------------------------------
	{
		"external entity declaration",
		`<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>` + planWith(`<text>&xxe;</text>`),
	},
	{
		"recursive entity expansion",
		`<!DOCTYPE svg [<!ENTITY a "aa"><!ENTITY b "&a;&a;">]>` + planWith(`<text>&b;</text>`),
	},
	{
		"not well-formed",
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900"><g data-floor="I-1301">`,
	},
}

func TestNormalizeRejectsHostileUploads(t *testing.T) {
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Normalize([]byte(tc.svg), Expect{FloorID: "I-1301"})
			if err == nil {
				t.Fatalf("Normalize() accepted it and produced:\n%s", plan.SVG)
			}
			// The admin has to be told what to fix, so these are client
			// errors, not masked 500s.
			if kind := apperr.KindOf(err); kind != apperr.KindInvalid {
				t.Errorf("KindOf(err) = %v, want KindInvalid (err: %v)", kind, err)
			}
		})
	}
}

// Some payloads are legitimate content that must survive as content. The
// risk is the opposite one: text that gets re-emitted as markup. <title> and
// <desc> are HTML integration points, so the app's HTML parser reads their
// contents as HTML when the plan is inlined.
func TestNormalizeNeutralizesTextWithoutRejectingIt(t *testing.T) {
	cases := map[string]string{
		"markup in a title": planWith(`<title>&lt;img src=x onerror=alert(1)&gt;</title>`),
		"markup in a desc":  planWith(`<desc>&lt;script&gt;alert(1)&lt;/script&gt;</desc>`),
		"markup in a label": planWith(`<text>&lt;b&gt;ห้องตรวจ&lt;/b&gt;</text>`),
		"quote in a label":  planWith(`<text data-place-id="REG-01">say &quot;hi&quot;</text>`),
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			plan, err := Normalize([]byte(src), Expect{FloorID: "I-1301"})
			if err != nil {
				t.Fatalf("Normalize() rejected ordinary text: %v", err)
			}
			assertInert(t, plan.SVG)
		})
	}
}

// Whatever the corpus does, anything Normalize hands back must be inert.
// This is the backstop for a payload nobody thought to write a case for:
// every input above is run through, and any that is accepted must come back
// clean.
func TestNormalizeOutputIsAlwaysInert(t *testing.T) {
	for _, tc := range rejected {
		plan, err := Normalize([]byte(tc.svg), Expect{FloorID: "I-1301"})
		if err != nil {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { assertInert(t, plan.SVG) })
	}
}

// assertInert checks that a serialized plan is a drawing and nothing more.
//
// It asks two different questions, because they fail differently. The
// structural one is the real proof: feeding the output back through
// Normalize only succeeds if every name in it survived the allowlist, so
// nothing can be hiding in there that parse would have refused. The textual
// one is a readable tripwire over the constructs a reviewer looks for.
//
// The markup checks run over tag spans only, on purpose. Escaped text is
// allowed to contain anything — "&lt;img src=x onerror=alert(1)&gt;" in a
// <title> is a room label that reads oddly, not a handler, and treating it
// as a finding would push the fix in exactly the wrong direction: towards
// deleting patient-visible text instead of escaping it.
func assertInert(t *testing.T, svg string) {
	t.Helper()

	if _, err := Normalize([]byte(svg), Expect{FloorID: "I-1301"}); err != nil {
		t.Errorf("normalized output is not itself acceptable, so it contains something parse refuses: %v", err)
	}

	lower := strings.ToLower(svg)
	for _, b := range []string{
		"<script", "<foreignobject", "<iframe", "<image", "<use", "<animate",
		"<!doctype", ":root", ":host", "@import",
		"javascript:", "url(http", "url(//", "url(data:",
	} {
		if strings.Contains(lower, b) {
			t.Errorf("normalized output contains %q:\n%s", b, svg)
		}
	}

	markup := strings.ToLower(tagSpans(svg))
	for _, b := range []string{"onload=", "onerror=", "onclick=", "xlink:"} {
		if strings.Contains(markup, b) {
			t.Errorf("normalized output has %q inside a tag:\n%s", b, svg)
		}
	}
}

// tagSpans returns only the parts of the document a browser reads as tags,
// so text content cannot trip an attribute check.
func tagSpans(svg string) string {
	var b strings.Builder
	for rest := svg; ; {
		open := strings.IndexByte(rest, '<')
		if open < 0 {
			return b.String()
		}
		rest = rest[open:]
		close := strings.IndexByte(rest, '>')
		if close < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:close+1])
		rest = rest[close+1:]
	}
}
