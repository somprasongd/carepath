package floorplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Normalize turns uploaded bytes into a trusted Plan, or rejects them.
//
// The pipeline is: cap the size, parse, keep only what the allowlist names,
// check the plan against what the map model already believes (Expect), then
// write the result out from the parsed tree. The returned SVG shares no
// bytes with the input — every name, attribute and character in it was
// emitted by this function.
func Normalize(src []byte, expect Expect) (Plan, error) {
	if len(src) > MaxBytes {
		return Plan{}, ErrTooLarge
	}

	root, err := parse(src)
	if err != nil {
		return Plan{}, err
	}
	if root.name != "svg" {
		return Plan{}, rejectf("the root element must be <svg>, not <%s>", truncate(root.name))
	}

	viewBox, err := checkViewBox(root, expect.ViewBox)
	if err != nil {
		return Plan{}, err
	}
	if err := checkFloorGroup(root, expect.FloorID); err != nil {
		return Plan{}, err
	}
	if !hasID(root, "route-layer") {
		return Plan{}, rejectf(`no element with id="route-layer" — the app draws the patient's route into that layer`)
	}

	svg := serialize(root, viewBox)
	digest := sha256.Sum256([]byte(svg))
	return Plan{
		SVG:      svg,
		SHA256:   hex.EncodeToString(digest[:]),
		ViewBox:  viewBox,
		Warnings: collectWarnings(root, expect),
	}, nil
}

// node is one element of the parsed plan, or a run of character data when
// name is empty. Only allowlisted names ever reach it.
type node struct {
	name  string
	attrs []xml.Attr
	kids  []*node
	text  string
}

// parse reads the upload into a tree of allowlisted nodes.
//
// Everything rejected here is rejected rather than stripped: the admin is at
// the keyboard and a silent edit would leave them with a plan that is not
// the one they drew.
func parse(src []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(src))
	dec.Strict = true

	var (
		root     *node
		stack    []*node
		elements int
		styleLen int
	)
	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrMalformed
		}

		switch t := token.(type) {
		case xml.Directive:
			// A DOCTYPE is the door to entity expansion and external entity
			// resolution. Plans have no use for one.
			return nil, rejectf("document type declarations are not allowed")

		case xml.ProcInst, xml.Comment:
			// The XML declaration and comments carry nothing we re-emit.

		case xml.StartElement:
			elements++
			if elements > MaxElements {
				return nil, rejectf("more than %d elements", MaxElements)
			}
			if t.Name.Space != SVGNamespace && t.Name.Space != "" {
				// Catches the trick of declaring an xhtml (or any other)
				// default namespace so a foreign element wears an
				// SVG-looking local name.
				return nil, rejectf("element <%s> is not in the SVG namespace", truncate(t.Name.Local))
			}
			if !allowedElements[t.Name.Local] {
				return nil, rejectf("element <%s> is not allowed", truncate(t.Name.Local))
			}
			attrs, err := cleanAttrs(t)
			if err != nil {
				return nil, err
			}
			n := &node{name: t.Name.Local, attrs: attrs}
			if root == nil {
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.kids = append(parent.kids, n)
			}
			stack = append(stack, n)

		case xml.EndElement:
			if len(stack) == 0 {
				return nil, ErrMalformed
			}
			stack = stack[:len(stack)-1]

		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			parent := stack[len(stack)-1]
			if !textElements[parent.name] {
				// Indentation between elements, and any stray text in a
				// container that would not render anyway.
				continue
			}
			text := string(t)
			if parent.name == "style" {
				styleLen += len(text)
				if styleLen > MaxStyleBytes {
					return nil, rejectf("more than %d bytes of CSS", MaxStyleBytes)
				}
				if text, err = cleanCSS(text); err != nil {
					return nil, err
				}
			}
			parent.kids = append(parent.kids, &node{text: text})
		}
	}

	if root == nil {
		return nil, ErrMalformed
	}
	return root, nil
}

// cleanAttrs keeps the allowlisted attributes of one element and checks the
// values that can reach a URL resolver or a CSS parser.
func cleanAttrs(el xml.StartElement) ([]xml.Attr, error) {
	attrs := make([]xml.Attr, 0, len(el.Attr))
	for _, attr := range el.Attr {
		switch {
		case attr.Name.Local == "xmlns" && attr.Name.Space == "":
			// Dropped here and re-emitted on the root by serialize, so the
			// output declares exactly one namespace and it is ours.
			continue
		case attr.Name.Space != "":
			// Namespaced attributes — xmlns:*, xlink:href, xml:* — carry
			// references or parser switches the plans do not need.
			return nil, rejectf("namespaced attribute %s: is not allowed on <%s>",
				truncate(attr.Name.Space), truncate(el.Name.Local))
		case !allowedAttr(attr.Name.Local):
			return nil, rejectf("attribute %q is not allowed on <%s>",
				truncate(attr.Name.Local), truncate(el.Name.Local))
		}

		if err := checkValue(attr.Name.Local, attr.Value); err != nil {
			return nil, err
		}
		if attr.Name.Local == "style" {
			value, err := cleanCSS(attr.Value)
			if err != nil {
				return nil, err
			}
			attr.Value = value
		}
		attrs = append(attrs, attr)
	}

	// Sorted so the same drawing always serializes to the same bytes, and
	// therefore to the same digest: re-uploading a file that only differs in
	// attribute order does not create a second stored plan.
	slices.SortFunc(attrs, func(a, b xml.Attr) int {
		return strings.Compare(a.Name.Local, b.Name.Local)
	})
	return attrs, nil
}

// checkValue rejects attribute values that could dereference something.
// The XML parser has already decoded character references, so what is
// checked here is what the browser would act on.
func checkValue(local, value string) error {
	if strings.Contains(strings.ToLower(squeeze(value)), "javascript:") {
		return rejectf("attribute %q contains a javascript: URL", truncate(local))
	}
	if err := checkURLRefs(value, "attribute "+strconv.Quote(truncate(local))); err != nil {
		return err
	}
	return nil
}

// checkURLRefs allows url(#fragment) — the plans point marker-end at their
// own arrow head that way — and nothing else. An external url() is a
// request the patient's browser would make to a third party the moment the
// plan renders.
func checkURLRefs(value, what string) error {
	rest := value
	for {
		// indexFoldASCII, not strings.Index(strings.ToLower(rest), ...): the
		// needle is pure ASCII, and strings.ToLower is not byte-length
		// preserving for every rune (e.g. "İ" 2 bytes -> "i̇" 3 bytes) — an
		// index found in a lowercased copy can land on the wrong byte once
		// used to slice the original string.
		i := indexFoldASCII(rest, "url(")
		if i < 0 {
			return nil
		}
		target := strings.TrimLeft(rest[i+len("url("):], " \t\r\n'\"")
		if !strings.HasPrefix(target, "#") {
			return rejectf("%s references something outside the file with url()", what)
		}
		rest = rest[i+len("url("):]
	}
}

// indexFoldASCII finds an all-lowercase ASCII needle in s, matching its
// ASCII letters case-insensitively. Unlike strings.Index(strings.ToLower(s),
// needle), it never touches non-ASCII bytes, so the returned index always
// refers to the same byte in s that a caller then slices.
func indexFoldASCII(s, needle string) int {
	for i := 0; i+len(needle) <= len(s); i++ {
		if asciiEqualFold(s[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

// asciiEqualFold reports whether a and b are equal, folding only ASCII
// letters (b is assumed already lowercase).
func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		c := a[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != b[i] {
			return false
		}
	}
	return true
}

// cleanCSS checks a <style> body or a style="" value and scopes it.
//
// The rewrite is the point: :root is the document root, so a plan inlined
// into the app would restyle the whole page through it. Rewritten to svg,
// the custom properties land on the plan's own root element and inherit
// down from there — the colours keep working, including inside a shadow
// root, where :root would have matched nothing at all.
func cleanCSS(css string) (string, error) {
	lower := strings.ToLower(css)
	// CSS needs neither of these, and both are how a style body escapes
	// itself: "</style><script>" written directly, or written as character
	// references for a parser that decodes them.
	if strings.ContainsAny(css, "<&") {
		return "", rejectf("CSS must not contain '<' or '&'")
	}
	for _, banned := range []string{"@import", "expression(", ":host", "::part", "::slotted"} {
		if strings.Contains(lower, banned) {
			return "", rejectf("CSS must not use %s", banned)
		}
	}
	if strings.Contains(squeezeLower(css), "javascript:") {
		return "", rejectf("CSS contains a javascript: URL")
	}
	if err := checkURLRefs(css, "CSS"); err != nil {
		return "", err
	}
	return strings.ReplaceAll(css, ":root", "svg"), nil
}

// checkViewBox reads the plan's coordinate space and holds it to the floor's
// established one. Every place.x/y and nav_node.x/y is a point in that
// space, so accepting a different one would move every pin and every route
// without a single error anywhere.
func checkViewBox(root *node, want string) (string, error) {
	raw, ok := attrValue(root, "viewBox")
	if !ok {
		return "", rejectf("the root <svg> has no viewBox")
	}
	got, err := canonicalViewBox(raw)
	if err != nil {
		return "", err
	}
	if want == "" {
		return got, nil
	}
	wantCanonical, err := canonicalViewBox(want)
	if err != nil {
		return "", err
	}
	if got != wantCanonical {
		return "", rejectf("viewBox is %q but this floor's plans are drawn in %q; "+
			"every place and navigation node is positioned in that space", got, wantCanonical)
	}
	return got, nil
}

// canonicalViewBox parses "minX minY width height" and reformats it so two
// spellings of the same space compare equal.
func canonicalViewBox(raw string) (string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) != 4 {
		return "", rejectf("viewBox %q must be four numbers", truncate(raw))
	}
	values := make([]string, 4)
	for i, field := range fields {
		f, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return "", rejectf("viewBox %q must be four numbers", truncate(raw))
		}
		if i >= 2 && f <= 0 {
			return "", rejectf("viewBox %q must have a positive width and height", truncate(raw))
		}
		values[i] = strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strings.Join(values, " "), nil
}

// checkFloorGroup holds the plan to exactly one floor group naming the floor
// being uploaded to. FloorPlanMap measures that group to fit the plan into
// the map card, and the app would otherwise happily show floor 2's drawing
// under floor 1's coordinates.
func checkFloorGroup(root *node, floorID string) error {
	// Must be a <g>: FloorPlanMap only ever queries `g[data-floor]` to find
	// the group it measures for fitting the plan into the map card. A
	// data-floor on any other element would pass a name/count check but
	// still be invisible to that query, so the plan would render unfitted
	// with no error anywhere.
	var found []string
	walk(root, func(n *node) {
		if value, ok := attrValue(n, "data-floor"); ok && n.name == "g" {
			found = append(found, value)
		}
	})
	switch {
	case len(found) == 0:
		return rejectf(`no <g> element carries data-floor="%s"`, truncate(floorID))
	case len(found) > 1:
		return rejectf("the plan declares %d floor groups; a plan draws exactly one floor", len(found))
	case found[0] != floorID:
		return rejectf("the plan is drawn for floor %q but was uploaded to floor %q",
			truncate(found[0]), truncate(floorID))
	}
	return nil
}

// collectWarnings reports what the plan and the map model disagree about
// without blocking the upload: a place the plan has not drawn yet is already
// an honest "not routable" state in the patient app, and refusing the whole
// plan over one room would be worse than showing the rest of the floor.
func collectWarnings(root *node, expect Expect) []Warning {
	drawnPlaces := make(map[string]bool)
	drawnIDs := make(map[string]bool)
	walk(root, func(n *node) {
		if value, ok := attrValue(n, "data-place-id"); ok {
			drawnPlaces[value] = true
		}
		if value, ok := attrValue(n, "id"); ok {
			drawnIDs[value] = true
		}
	})

	warnings := make([]Warning, 0)
	for _, placeID := range expect.PlaceIDs {
		if !drawnPlaces[placeID] {
			warnings = append(warnings, Warning{Code: WarnPlaceMissing, Ref: placeID})
		}
	}
	known := make(map[string]bool, len(expect.PlaceIDs))
	for _, placeID := range expect.PlaceIDs {
		known[placeID] = true
	}
	for placeID := range drawnPlaces {
		if !known[placeID] {
			warnings = append(warnings, Warning{Code: WarnPlaceUnknown, Ref: placeID})
		}
	}
	for _, nodeID := range expect.NodeIDs {
		if !drawnIDs[nodeID] {
			warnings = append(warnings, Warning{Code: WarnNodeMissing, Ref: nodeID})
		}
	}

	// Map iteration makes PLACE_UNKNOWN order random; sort so the same
	// upload always reports the same list.
	slices.SortFunc(warnings, func(a, b Warning) int {
		if c := strings.Compare(a.Code, b.Code); c != 0 {
			return c
		}
		return strings.Compare(a.Ref, b.Ref)
	})
	return warnings
}

// serialize writes the tree back out. This is the step that makes the result
// trusted: it can only emit names that survived the allowlist, and it does
// its own escaping, so no byte of the upload passes through as markup.
func serialize(root *node, viewBox string) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="` + SVGNamespace + `"`)
	for _, attr := range root.attrs {
		if attr.Name.Local == "viewBox" {
			continue
		}
		writeAttr(&b, attr.Name.Local, attr.Value)
	}
	writeAttr(&b, "viewBox", viewBox)
	b.WriteString(">")
	for _, kid := range root.kids {
		writeNode(&b, kid)
	}
	b.WriteString("</svg>")
	return b.String()
}

func writeNode(b *strings.Builder, n *node) {
	if n.name == "" {
		// A <style> body is written as-is: cleanCSS has already refused '<'
		// and '&', so there is nothing left to escape — and escaping it
		// would corrupt CSS that a browser reads as raw text.
		b.WriteString(n.text)
		return
	}
	b.WriteByte('<')
	b.WriteString(n.name)
	for _, attr := range n.attrs {
		writeAttr(b, attr.Name.Local, attr.Value)
	}
	if len(n.kids) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteString(">")
	for _, kid := range n.kids {
		if kid.name == "" && !textElements[n.name] {
			continue
		}
		if kid.name == "" && n.name != "style" {
			b.WriteString(escapeText(kid.text))
			continue
		}
		writeNode(b, kid)
	}
	b.WriteString("</")
	b.WriteString(n.name)
	b.WriteByte('>')
}

func writeAttr(b *strings.Builder, name, value string) {
	b.WriteByte(' ')
	b.WriteString(name)
	b.WriteString(`="`)
	b.WriteString(escapeAttr(value))
	b.WriteByte('"')
}

// escapeText escapes the characters that would otherwise start markup.
// <title> and <desc> are HTML integration points, so their text is parsed as
// HTML when the app inlines the plan — unescaped text there is markup.
// textReplacer and attrReplacer are package-level rather than built per call:
// a plan runs to hundreds of attributes, and each was allocating its own
// Replacer and trie for the same fixed substitution table.
var (
	textReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	attrReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
)

func escapeText(s string) string {
	return textReplacer.Replace(s)
}

func escapeAttr(s string) string {
	return attrReplacer.Replace(s)
}

func attrValue(n *node, local string) (string, bool) {
	for _, attr := range n.attrs {
		if attr.Name.Local == local {
			return attr.Value, true
		}
	}
	return "", false
}

func hasID(root *node, id string) bool {
	found := false
	walk(root, func(n *node) {
		if value, ok := attrValue(n, "id"); ok && value == id {
			found = true
		}
	})
	return found
}

func walk(n *node, fn func(*node)) {
	if n.name != "" {
		fn(n)
	}
	for _, kid := range n.kids {
		walk(kid, fn)
	}
}

// squeeze removes the whitespace and control characters a browser ignores
// inside a URL, so "java\tscript:" is checked as "javascript:".
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

func squeezeLower(s string) string { return strings.ToLower(squeeze(s)) }

// truncate bounds what a rejection message echoes back from the upload.
func truncate(s string) string {
	const max = 40
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
