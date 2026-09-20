import { useCallback, useEffect, useRef, useState } from 'react'

export type ViewBox = { x: number; y: number; width: number; height: number }

/** Parse an SVG `viewBox` attribute; null when absent or malformed. */
export function parseViewBox(value: string | null): ViewBox | null {
  if (!value) return null
  const parts = value.trim().split(/[\s,]+/).map(Number)
  if (
    parts.length !== 4 ||
    parts.some((n) => !Number.isFinite(n)) ||
    parts[2] <= 0 ||
    parts[3] <= 0
  ) {
    return null
  }
  return { x: parts[0], y: parts[1], width: parts[2], height: parts[3] }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

/**
 * The window the map focuses on around the destination: 640 SVG units wide
 * (or the whole plan when narrower) keeps room labels readable at a phone's
 * map width. `aspect` (width/height) is the rendered container's, so a tall
 * phone map area is filled edge to edge instead of letterboxing a 16:9 strip;
 * clamping inside the base viewBox means the focus never shows space outside
 * the floor (#25 AC4).
 */
export function focusedViewBox(
  base: ViewBox,
  center: { x: number; y: number },
  aspect?: number,
): ViewBox {
  const shape =
    aspect && Number.isFinite(aspect) && aspect > 0 ? aspect : base.width / base.height
  let width = Math.min(base.width, 640)
  let height = width / shape
  if (height > base.height) {
    height = base.height
    width = height * shape
  }
  return {
    x: clamp(center.x - width / 2, base.x, base.x + base.width - width),
    y: clamp(center.y - height / 2, base.y, base.y + base.height - height),
    width,
    height,
  }
}

export type FloorPlanDestination = {
  /** Stable SVG place id — the join key to data-place-id in the plan. */
  placeId: string
  /** Patient-facing name drawn in the pin's label. */
  name: string
  /** Floor-local SVG units of the place's entry node; falls back to the room's centre. */
  x?: number
  y?: number
}

/**
 * The walking line(s) to draw on this floor's plan (#29) — plain geometry
 * in SVG units; how it is derived from a route lives in features/navigation.
 * Each line is drawn with the asset's own route classes so the overlay and
 * the plan read as one system, with a route-dot marking where the patient
 * stands (the first point of the first line). `origin` (#35) upgrades that
 * dot to a labelled "you are here" mark in the secondary colour — pass it
 * only when the patient really is on the displayed floor.
 */
export type FloorPlanRoute = {
  lines: { x: number; y: number }[][]
  origin?: { x: number; y: number }
}

export type FloorPlanMapProps = {
  /** Raw SVG markup of a packages/floorplans floor plan. */
  svg: string
  /** Patient-facing floor label for the map caption, e.g. "Floor 1". */
  floorLabel: string
  destination?: FloorPlanDestination
  route?: FloorPlanRoute
  /** Spoken label for the whole map; the caller composes it translated. */
  ariaLabel?: string
  /**
   * Button and you-are-here captions, supplied translated (ADR-0012). The
   * focus toggle is hidden and the origin mark drops its label without them.
   */
  labels?: {
    viewFullFloor: string
    viewRoute: string
    viewDestination: string
    youAreHere: string
    zoomIn: string
    zoomOut: string
  }
}

/**
 * The window the map focuses on around the route (#29 AC4): wide enough to
 * keep the whole walking line plus its origin dot on screen at phone width
 * (never smaller than the destination focus's 640 units, so room labels stay
 * readable), clamped inside the base viewBox so the focus never shows space
 * outside the floor. `bounds` is the line's bbox as two corner points.
 */
export function routedViewBox(
  base: ViewBox,
  bounds: { from: { x: number; y: number }; to: { x: number; y: number } },
  aspect?: number,
): ViewBox {
  const shape =
    aspect && Number.isFinite(aspect) && aspect > 0 ? aspect : base.width / base.height
  const pad = 90
  const width = Math.min(base.width, Math.max(640, bounds.to.x - bounds.from.x + pad * 2))
  // Fill the container's shape when the floor allows it, but never trade the
  // width above for height: a tall phone map area would otherwise shrink the
  // window below the readability floor and clip the line off-screen.
  const height = Math.min(
    base.height,
    Math.max(width / shape, bounds.to.y - bounds.from.y + pad * 2),
  )
  const center = {
    x: (bounds.from.x + bounds.to.x) / 2,
    y: (bounds.from.y + bounds.to.y) / 2,
  }
  return {
    x: clamp(center.x - width / 2, base.x, base.x + base.width - width),
    y: clamp(center.y - height / 2, base.y, base.y + base.height - height),
    width,
    height,
  }
}

const SVG_NS = 'http://www.w3.org/2000/svg'

/** How far a free zoom may go in beyond the whole-plan view (8×). */
export const MAX_ZOOM = 8

/** Breathing room around the floor group when fitting "the whole floor". */
const FIT_PAD = 20

/** How much of the window must stay over the plan when free-panning (25%). */
const KEEP_VISIBLE = 0.25

/**
 * Keep a view sane over the base plan after a free gesture: never larger
 * than the whole plan, and positioned so the plan never leaves the window —
 * an axis where the plan is zoomed in stays edge-locked, but an axis where
 * the plan exactly fits (fully zoomed out) may drift, because the map area's
 * lower part is usually covered by the bottom sheet: sliding the plan up
 * under it is the only way to read what the sheet hides.
 */
export function clampView(view: ViewBox, base: ViewBox): ViewBox {
  const clamped = { ...view }
  if (clamped.width > base.width) {
    clamped.height *= base.width / clamped.width
    clamped.width = base.width
  }
  if (clamped.height > base.height) {
    clamped.width *= base.height / clamped.height
    clamped.height = base.height
  }
  const clampAxis = (pos: number, size: number, basePos: number, baseSize: number) =>
    size >= baseSize
      ? clamp(pos, basePos - (1 - KEEP_VISIBLE) * size, basePos + baseSize - KEEP_VISIBLE * size)
      : clamp(pos, basePos, basePos + baseSize - size)
  clamped.x = clampAxis(clamped.x, clamped.width, base.x, base.width)
  clamped.y = clampAxis(clamped.y, clamped.height, base.y, base.height)
  return clamped
}

/**
 * Zoom by `factor` (<1 in, >1 out) keeping the viewport point at ratios
 * (rx, ry in 0..1 — where the cursor or pinch midpoint sits) pinned to the
 * same plan point, then clamp. The view's own aspect ratio is preserved:
 * the host box does not change shape while zooming.
 */
export function zoomedView(
  view: ViewBox,
  base: ViewBox,
  factor: number,
  rx = 0.5,
  ry = 0.5,
): ViewBox {
  const ratio = view.height / view.width
  let width = view.width * factor
  if (width * ratio > base.height) width = base.height / ratio
  width = clamp(width, base.width / MAX_ZOOM, base.width)
  const height = width * ratio
  return clampView(
    {
      x: view.x + rx * view.width - rx * width,
      y: view.y + ry * view.height - ry * height,
      width,
      height,
    },
    base,
  )
}

/**
 * The real floor-plan asset (packages/floorplans), inlined so its DOM stays
 * reachable: the destination room gets the route-colour edge, a pin in the
 * same shape language as the asset's own route dots plus a name label, and
 * the view opens focused on the route (or the destination before any fix).
 * The whole floor is one toggle away, and from there the patient owns the
 * view: drag pans, wheel/pinch zooms toward the fingers, double-tap resets.
 * A manual view survives re-renders of the same floor and resets when the
 * floor changes.
 */
export function FloorPlanMap({ svg, floorLabel, destination, route, ariaLabel, labels }: FloorPlanMapProps) {
  const hostRef = useRef<HTMLDivElement>(null)
  const svgElRef = useRef<SVGSVGElement | null>(null)
  const baseRef = useRef<ViewBox | null>(null)
  const viewRef = useRef<ViewBox | null>(null)
  // Which asset the current custom view belongs to — a floor switch resets.
  const customForRef = useRef<string | null>(null)
  const [mode, setMode] = useState<'auto' | 'full' | 'custom'>('auto')

  const applyView = useCallback(() => {
    const view = viewRef.current
    if (svgElRef.current && view) {
      svgElRef.current.setAttribute(
        'viewBox',
        `${view.x} ${view.y} ${view.width} ${view.height}`,
      )
    }
  }, [])

  const commit = useCallback(() => {
    if (baseRef.current && viewRef.current) {
      viewRef.current = clampView(viewRef.current, baseRef.current)
    }
    applyView()
  }, [applyView])
  const becomeCustom = useCallback(() => {
    customForRef.current = svg
    setMode((current) => (current === 'custom' ? current : 'custom'))
  }, [svg])
  /** Step zoom around the map's centre — the +/− buttons' handler. */
  const zoomAround = useCallback(
    (factor: number) => {
      const base = baseRef.current
      const view = viewRef.current
      if (!base || !view) return
      viewRef.current = zoomedView(view, base, factor, 0.5, 0.5)
      commit()
      becomeCustom()
    },
    [commit, becomeCustom],
  )

  useEffect(() => {
    const host = hostRef.current
    if (!host) return

    // Re-inject on every dependent change: the asset is static markup, so
    // clearing this component's own DOM edits by resetting innerHTML is
    // simpler than tracking and undoing each one.
    host.innerHTML = svg

    const el = host.querySelector('svg')
    if (!el) return
    svgElRef.current = el
    el.style.width = '100%'
    el.style.height = 'auto'
    el.style.display = 'block'

    const base = parseViewBox(el.getAttribute('viewBox')) ?? {
      x: 0,
      y: 0,
      width: 1600,
      height: 900,
    }

    // The asset canvas carries strips the floor itself doesn't — a title
    // block, a legend, blank margins around the translated floor group.
    // "The whole floor" (and every pan/zoom clamp) is the floor group's own
    // bounds, measured in viewBox coordinates, so the plan fills the map
    // card instead of letterboxing inside its own whitespace.
    const floorGroup = el.querySelector('g[data-floor]')
    const floorBounds =
      floorGroup instanceof SVGGElement ? absoluteBox(el, floorGroup) : null
    const fitView = floorBounds
      ? clampView(
          {
            x: floorBounds.from.x - FIT_PAD,
            y: floorBounds.from.y - FIT_PAD,
            width: floorBounds.to.x - floorBounds.from.x + FIT_PAD * 2,
            height: floorBounds.to.y - floorBounds.from.y + FIT_PAD * 2,
          },
          base,
        )
      : base
    baseRef.current = fitView

    // The asset ships a sample route hidden by default — keep it that way;
    // the live route is drawn fresh below, never by editing the asset.
    el.querySelector('#route-layer')?.setAttribute('opacity', '0')

    const floor = el.querySelector('g[data-floor]')
    const center =
      destination && floor
        ? drawDestination(floor as SVGGElement, el, destination)
        : null

    let routeLinesBounds: { from: { x: number; y: number }; to: { x: number; y: number } } | null =
      null
    if (route && floor) {
      const drawn = drawRoute(floor as SVGGElement, route, labels?.youAreHere)
      routeLinesBounds = drawn ? absoluteBox(el, drawn) : null
    }

    // Measured so the focused window fills the actual map area — on a phone
    // that area is taller than wide, and a plan-shaped window would letterbox.
    const hostRect = host.getBoundingClientRect()
    const containerAspect =
      hostRect.width > 0 && hostRect.height > 0 ? hostRect.width / hostRect.height : undefined
    // A manual view survives re-renders of the same floor (the patient may
    // be mid-inspection when the route or destination refetches); a floor
    // switch or a toggle fall back to the computed view.
    const keepCustom = mode === 'custom' && viewRef.current && customForRef.current === svg
    if (!keepCustom) {
      customForRef.current = null
      viewRef.current =
        mode === 'full'
          ? fitView
          : routeLinesBounds
            ? routedViewBox(base, routeLinesBounds, containerAspect)
            : center
              ? focusedViewBox(base, center, containerAspect)
              : base
    }
    applyView()

    if (destination && ariaLabel) {
      el.setAttribute('aria-label', ariaLabel)
    }
  }, [svg, destination, floorLabel, mode, route, ariaLabel, labels, applyView])

  // Free pan/zoom over the injected plan (#25 follow-up): one pointer drags,
  // two pointers pinch (the midpoint stays put), the wheel zooms toward the
  // cursor. Gestures write the viewBox directly — React state only records
  // that the view became custom, so panning stays 60fps. Bound to the host
  // (not React props) because wheel needs a non-passive listener and the
  // pointer set lives across events.
  useEffect(() => {
    const host = hostRef.current
    if (!host) return

    const pointers = new Map<number, { x: number; y: number }>()
    let pinchDist = 0

    const zoomAt = (cx: number, cy: number, factor: number) => {
      const rect = host.getBoundingClientRect()
      const base = baseRef.current
      const view = viewRef.current
      if (!base || !view || rect.width === 0) return
      viewRef.current = zoomedView(
        view,
        base,
        factor,
        (cx - rect.left) / rect.width,
        (cy - rect.top) / rect.height,
      )
    }

    const onWheel = (event: WheelEvent) => {
      event.preventDefault()
      zoomAt(event.clientX, event.clientY, Math.exp(event.deltaY * 0.0015))
      commit()
      becomeCustom()
    }
    const onPointerDown = (event: PointerEvent) => {
      pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
      pinchDist = 0
      try {
        host.setPointerCapture(event.pointerId)
      } catch {
        // A pointer that cannot be captured (already released, or a
        // synthetic event) still pans — its moves arrive on the host either
        // way while the finger is over it.
      }
    }
    const onPointerMove = (event: PointerEvent) => {
      const prev = pointers.get(event.pointerId)
      if (!prev) return
      const view = viewRef.current
      const rect = host.getBoundingClientRect()
      if (!view || rect.width === 0) return

      if (pointers.size === 1) {
        view.x -= ((event.clientX - prev.x) * view.width) / rect.width
        view.y -= ((event.clientY - prev.y) * view.height) / rect.height
        commit()
        becomeCustom()
      }

      pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })

      if (pointers.size === 2) {
        const [a, b] = Array.from(pointers.values())
        const dist = Math.hypot(a.x - b.x, a.y - b.y)
        // Fingers apart → dist grows → factor < 1 → zoom in, at the midpoint.
        if (pinchDist > 0 && dist > 0) {
          zoomAt((a.x + b.x) / 2, (a.y + b.y) / 2, pinchDist / dist)
          commit()
          becomeCustom()
        }
        pinchDist = dist
      }
    }
    const onPointerEnd = (event: PointerEvent) => {
      pointers.delete(event.pointerId)
      pinchDist = 0
    }
    const onDoubleClick = () => setMode('auto')

    host.addEventListener('wheel', onWheel, { passive: false })
    host.addEventListener('pointerdown', onPointerDown)
    host.addEventListener('pointermove', onPointerMove)
    host.addEventListener('pointerup', onPointerEnd)
    host.addEventListener('pointercancel', onPointerEnd)
    host.addEventListener('dblclick', onDoubleClick)
    return () => {
      host.removeEventListener('wheel', onWheel)
      host.removeEventListener('pointerdown', onPointerDown)
      host.removeEventListener('pointermove', onPointerMove)
      host.removeEventListener('pointerup', onPointerEnd)
      host.removeEventListener('pointercancel', onPointerEnd)
      host.removeEventListener('dblclick', onDoubleClick)
    }
  }, [svg, commit, becomeCustom])

  return (
    <div className="relative h-full w-full overflow-hidden rounded-lg border border-line bg-surface">
      {/* The plan is a repo-controlled static asset, not user input — inlining
          it is what keeps the destination highlight and pin reachable. */}
      <div
        ref={hostRef}
        className="flex h-full w-full touch-none cursor-grab select-none items-center justify-center active:cursor-grabbing [&>svg]:m-auto"
        dangerouslySetInnerHTML={{ __html: svg }}
      />
      {/* Right-edge stack: the view toggle above the step zoom controls —
          pinch/wheel zoom is undiscoverable on a desktop, and panning needs
          zoom first (the auto view already shows the plan's full height). */}
      <div className="absolute top-2.5 right-2.5 z-10 flex flex-col gap-1.5">
        {labels && (destination || route) && (
          <button
            type="button"
            onClick={() => setMode(mode === 'full' ? 'auto' : 'full')}
            className="cursor-pointer rounded-full border border-line bg-surface px-3 py-1.5 font-sans text-caption font-bold text-ink"
          >
            {mode === 'full'
              ? route
                ? labels.viewRoute
                : labels.viewDestination
              : labels.viewFullFloor}
          </button>
        )}
        {labels && (
          <button
            type="button"
            aria-label={labels.zoomIn}
            onClick={() => zoomAround(1 / 1.6)}
            className="cursor-pointer rounded-full border border-line bg-surface px-3 py-1.5 font-sans text-caption font-bold text-ink"
          >
            ＋
          </button>
        )}
        {labels && (
          <button
            type="button"
            aria-label={labels.zoomOut}
            onClick={() => zoomAround(1.6)}
            className="cursor-pointer rounded-full border border-line bg-surface px-3 py-1.5 font-sans text-caption font-bold text-ink"
          >
            −
          </button>
        )}
      </div>
      <span className="absolute bottom-2.5 left-2.5 rounded-full border border-line bg-surface px-2.5 py-1 font-sans text-caption font-bold text-ink-muted">
        {floorLabel}
      </span>
    </div>
  )
}

/** Quotes/backslashes escaped so a place id can travel inside an attribute selector. */
function attrSelectorValue(value: string): string {
  return value.replace(/["\\]/g, '\\$&')
}

function drawDestination(
  floor: SVGGElement,
  svg: SVGSVGElement,
  dest: FloorPlanDestination,
): { x: number; y: number } | null {
  // Rooms and nav nodes both carry data-place-id; the room rect is the one to outline.
  const room = Array.from(
    floor.querySelectorAll(`[data-place-id="${attrSelectorValue(dest.placeId)}"]`),
  ).find((node) => node.tagName.toLowerCase() === 'rect')
  if (room instanceof SVGRectElement) {
    const style = room.getAttribute('style') ?? ''
    room.setAttribute('style', `${style};stroke:var(--primary);stroke-width:6`)
  }

  const pin =
    typeof dest.x === 'number' && typeof dest.y === 'number'
      ? { x: dest.x, y: dest.y }
      : bboxCenter(room)
  if (!pin) return null

  return absoluteCenter(svg, appendPin(floor, pin, dest.name))
}

function bboxCenter(el: Element | undefined): { x: number; y: number } | null {
  if (!(el instanceof SVGGraphicsElement)) return null
  const box = el.getBBox()
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
}

/**
 * The pin borrows the asset's own shape language (route-dot circles, signage
 * label faces) so the app and the plan read as one system — see DESIGN.md.
 * Coordinates are floor-local, so it is appended inside the floor group.
 */
function appendPin(floor: SVGGElement, pin: { x: number; y: number }, name: string): SVGGElement {
  const g = document.createElementNS(SVG_NS, 'g')
  g.setAttribute('transform', `translate(${pin.x} ${pin.y})`)

  const halo = document.createElementNS(SVG_NS, 'circle')
  halo.setAttribute('r', '30')
  halo.setAttribute('fill', 'var(--primary)')
  halo.setAttribute('opacity', '0.18')
  const dot = document.createElementNS(SVG_NS, 'circle')
  dot.setAttribute('r', '12')
  dot.setAttribute('fill', 'var(--primary)')
  dot.setAttribute('stroke', 'var(--surface)')
  dot.setAttribute('stroke-width', '6')
  g.append(halo, dot)

  const labelHeight = 38
  const labelWidth = Math.max(120, name.length * 12 + 28)
  const label = document.createElementNS(SVG_NS, 'g')
  label.setAttribute('transform', 'translate(0 40)')
  const background = document.createElementNS(SVG_NS, 'rect')
  background.setAttribute('x', String(-labelWidth / 2))
  background.setAttribute('width', String(labelWidth))
  background.setAttribute('height', String(labelHeight))
  background.setAttribute('rx', '10')
  background.setAttribute('fill', 'var(--surface)')
  background.setAttribute('stroke', 'var(--primary-edge)')
  background.setAttribute('stroke-width', '2')
  const text = document.createElementNS(SVG_NS, 'text')
  text.setAttribute('y', String(labelHeight / 2))
  text.setAttribute('text-anchor', 'middle')
  text.setAttribute('dominant-baseline', 'central')
  text.setAttribute('font-family', "'Noto Sans Thai', Tahoma, sans-serif")
  text.setAttribute('font-size', '22')
  text.setAttribute('font-weight', '700')
  text.setAttribute('fill', 'var(--ink)')
  text.textContent = name
  label.append(background, text)
  g.append(label)

  floor.append(g)
  return g
}

/**
 * The walking line(s) for this floor, appended into the floor group so
 * floor-local coordinates land correctly. Each line is three layers: the
 * asset's `.route` solid stroke (rounded orange caps, dotted endpoints);
 * one direction arrow at the line's end — the asset's own `#arrow` marker
 * (orient=auto) pointing into the destination, carried on a stroke-less
 * polyline whose thin stroke-width keeps the marker wayfinding-sized
 * (markers scale with the referencing element's stroke, and the solid base
 * is 12 units wide); and `route-flow`, white dashes animated
 * origin→destination in index.css — the motion direction matches the point
 * order, which is the patient's walking order. The sample `#route-layer`
 * stays hidden; this group is the live one. Where the patient stands is a
 * plain route-dot, or — when the caller passes an origin and its label —
 * the labelled you-are-here mark from appendOriginMark.
 */
function drawRoute(
  floor: SVGGElement,
  route: FloorPlanRoute,
  youAreHere?: string,
): SVGGElement | null {
  const lines = route.lines.filter((line) => line.length > 0)
  if (lines.length === 0) return null

  const g = document.createElementNS(SVG_NS, 'g')
  for (const line of lines) {
    const points = line.map((p) => `${p.x},${p.y}`).join(' ')

    const polyline = document.createElementNS(SVG_NS, 'polyline')
    polyline.setAttribute('class', 'route')
    polyline.setAttribute('points', points)
    g.append(polyline)

    const arrow = document.createElementNS(SVG_NS, 'polyline')
    arrow.setAttribute('points', points)
    arrow.setAttribute('fill', 'none')
    arrow.setAttribute('stroke', 'none')
    arrow.setAttribute('stroke-width', '5')
    arrow.setAttribute('marker-end', 'url(#arrow)')
    g.append(arrow)

    const flow = document.createElementNS(SVG_NS, 'polyline')
    flow.setAttribute('class', 'route-flow')
    flow.setAttribute('points', points)
    flow.setAttribute('fill', 'none')
    flow.setAttribute('stroke', 'var(--surface)')
    flow.setAttribute('stroke-width', '6')
    flow.setAttribute('stroke-linecap', 'round')
    g.append(flow)
  }

  const origin = route.origin ?? lines[0][0]
  if (route.origin && youAreHere) {
    g.append(originMark(route.origin, youAreHere))
  } else {
    const dot = document.createElementNS(SVG_NS, 'circle')
    dot.setAttribute('class', 'route-dot')
    dot.setAttribute('cx', String(origin.x))
    dot.setAttribute('cy', String(origin.y))
    dot.setAttribute('r', '10')
    g.append(dot)
  }

  floor.append(g)
  return g
}

/**
 * The patient's position (#35): the destination pin's shape language in the
 * secondary colour — halo + dot + label above the point, so the you-are-here
 * label never collides with a destination label below it.
 */
function originMark(point: { x: number; y: number }, labelText: string): SVGGElement {
  const g = document.createElementNS(SVG_NS, 'g')
  g.setAttribute('transform', `translate(${point.x} ${point.y})`)

  const halo = document.createElementNS(SVG_NS, 'circle')
  halo.setAttribute('r', '30')
  halo.setAttribute('fill', 'var(--secondary)')
  halo.setAttribute('opacity', '0.18')
  const dot = document.createElementNS(SVG_NS, 'circle')
  dot.setAttribute('r', '12')
  dot.setAttribute('fill', 'var(--secondary)')
  dot.setAttribute('stroke', 'var(--surface)')
  dot.setAttribute('stroke-width', '6')
  g.append(halo, dot)

  const labelHeight = 34
  const labelWidth = Math.max(120, labelText.length * 12 + 28)
  const box = document.createElementNS(SVG_NS, 'g')
  box.setAttribute('transform', `translate(0 ${-40 - labelHeight})`)
  const background = document.createElementNS(SVG_NS, 'rect')
  background.setAttribute('x', String(-labelWidth / 2))
  background.setAttribute('width', String(labelWidth))
  background.setAttribute('height', String(labelHeight))
  background.setAttribute('rx', '10')
  background.setAttribute('fill', 'var(--surface)')
  background.setAttribute('stroke', 'var(--secondary)')
  background.setAttribute('stroke-width', '2')
  const text = document.createElementNS(SVG_NS, 'text')
  text.setAttribute('y', String(labelHeight / 2))
  text.setAttribute('text-anchor', 'middle')
  text.setAttribute('dominant-baseline', 'central')
  text.setAttribute('font-family', "'Noto Sans Thai', Tahoma, sans-serif")
  text.setAttribute('font-size', '22')
  text.setAttribute('font-weight', '700')
  text.setAttribute('fill', 'var(--ink)')
  text.textContent = labelText
  box.append(background, text)
  g.append(box)
  return g
}

/** A floor group's translate would shift route coordinates; measuring the rendered group instead is transform-proof. */
function absoluteBox(
  svg: SVGSVGElement,
  el: SVGGraphicsElement,
): { from: { x: number; y: number }; to: { x: number; y: number } } | null {
  const base = parseViewBox(svg.getAttribute('viewBox'))
  const svgRect = svg.getBoundingClientRect()
  if (!base || svgRect.width === 0) return null
  const rect = el.getBoundingClientRect()
  const scale = base.width / svgRect.width
  return {
    from: {
      x: base.x + (rect.left - svgRect.left) * scale,
      y: base.y + (rect.top - svgRect.top) * scale,
    },
    to: {
      x: base.x + (rect.right - svgRect.left) * scale,
      y: base.y + (rect.bottom - svgRect.top) * scale,
    },
  }
}

/** A floor group's translate would shift pin coordinates; measuring the rendered pin instead is transform-proof. */
function absoluteCenter(
  svg: SVGSVGElement,
  el: SVGGraphicsElement,
): { x: number; y: number } | null {
  const base = parseViewBox(svg.getAttribute('viewBox'))
  const svgRect = svg.getBoundingClientRect()
  if (!base || svgRect.width === 0) return null
  const rect = el.getBoundingClientRect()
  const scale = base.width / svgRect.width
  return {
    x: base.x + (rect.left + rect.width / 2 - svgRect.left) * scale,
    y: base.y + (rect.top + rect.height / 2 - svgRect.top) * scale,
  }
}
