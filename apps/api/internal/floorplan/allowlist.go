package floorplan

import "strings"

// allowedElements is every element a floor plan may contain. It is closed on
// purpose: anything an SVG editor emits that is not here has to be added
// deliberately, with a reason.
//
// Absent by design:
//   - script, foreignObject, iframe, embed, object, audio, video — script or
//     HTML execution contexts.
//   - image, use — they carry references; the plans are self-contained
//     schematics (ADR-0003) and nothing needs them yet.
//   - animate, set, animateTransform — SMIL can set attributes after the
//     allowlist has already run.
//   - filter, mask, pattern, and the gradients — unused by the MVP plans;
//     each one adds url() surface for no current gain.
var allowedElements = map[string]bool{
	"svg":      true,
	"g":        true,
	"defs":     true,
	"style":    true,
	"title":    true,
	"desc":     true,
	"marker":   true,
	"path":     true,
	"rect":     true,
	"circle":   true,
	"ellipse":  true,
	"line":     true,
	"polyline": true,
	"polygon":  true,
	"text":     true,
	"tspan":    true,
}

// textElements are the elements whose character data is content rather than
// layout whitespace. Everything else's text is dropped on the way out.
var textElements = map[string]bool{
	"style": true,
	"title": true,
	"desc":  true,
	"text":  true,
	"tspan": true,
}

// allowedAttrs is every attribute a floor plan may carry, by local name.
// data-* and aria-* pass through the prefix rule in allowedAttr instead —
// the plan conventions in packages/floorplans/README.md are built on
// data-floor, data-place-id, data-zone and data-node-type.
//
// The set is closed, which is what keeps every on* handler out without
// needing to enumerate them: onload, onerror and their several hundred
// siblings are simply not in here.
var allowedAttrs = map[string]bool{
	// identity and styling
	"id": true, "class": true, "style": true, "transform": true, "role": true,
	// paint
	"fill": true, "fill-opacity": true, "fill-rule": true,
	"stroke": true, "stroke-width": true, "stroke-linecap": true,
	"stroke-linejoin": true, "stroke-dasharray": true, "stroke-dashoffset": true,
	"stroke-opacity": true, "stroke-miterlimit": true,
	"opacity": true, "color": true, "vector-effect": true,
	"visibility": true, "display": true, "pointer-events": true, "clip-rule": true,
	// text
	"font-family": true, "font-size": true, "font-weight": true, "font-style": true,
	"text-anchor": true, "dominant-baseline": true, "alignment-baseline": true,
	"letter-spacing": true, "word-spacing": true, "dx": true, "dy": true,
	// geometry
	"x": true, "y": true, "width": true, "height": true,
	"rx": true, "ry": true, "cx": true, "cy": true, "r": true,
	"x1": true, "y1": true, "x2": true, "y2": true,
	"d": true, "points": true, "pathLength": true,
	// root
	"viewBox": true, "xmlns": true, "preserveAspectRatio": true,
	// markers
	"marker-start": true, "marker-mid": true, "marker-end": true,
	"markerWidth": true, "markerHeight": true, "markerUnits": true,
	"refX": true, "refY": true, "orient": true, "overflow": true,
}

// allowedAttr reports whether an attribute may be kept. data-* is open
// because the plan↔model join keys live there; aria-* is open because the
// plans carry their own accessible labelling (role, aria-labelledby) and
// stripping it would make the map worse for screen readers.
func allowedAttr(local string) bool {
	if strings.HasPrefix(local, "data-") || strings.HasPrefix(local, "aria-") {
		return true
	}
	return allowedAttrs[local]
}
