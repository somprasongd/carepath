import { useEffect, useRef, useState } from 'react'

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

export type FloorPlanMapProps = {
  /** Raw SVG markup of a packages/floorplans floor plan. */
  svg: string
  /** Thai floor label for the map caption, e.g. "ชั้น 1". */
  floorLabel: string
  destination?: FloorPlanDestination
}

const SVG_NS = 'http://www.w3.org/2000/svg'

/**
 * The real floor-plan asset (packages/floorplans), inlined so its DOM stays
 * reachable: the destination room gets the route-colour edge, a pin in the
 * same shape language as the asset's own route dots plus a name label, and
 * the view opens focused on it — the full-floor overview is one toggle away.
 * No route is drawn; turn-by-turn lines arrive with the navigation API
 * (#28/#29), never faked here.
 */
export function FloorPlanMap({ svg, floorLabel, destination }: FloorPlanMapProps) {
  const hostRef = useRef<HTMLDivElement>(null)
  const [focused, setFocused] = useState(true)

  useEffect(() => {
    const host = hostRef.current
    if (!host) return

    // Re-inject on every dependent change: the asset is static markup, so
    // clearing this component's own DOM edits by resetting innerHTML is
    // simpler than tracking and undoing each one.
    host.innerHTML = svg

    const el = host.querySelector('svg')
    if (!el) return
    el.style.width = '100%'
    el.style.height = 'auto'
    el.style.display = 'block'

    const base = parseViewBox(el.getAttribute('viewBox')) ?? {
      x: 0,
      y: 0,
      width: 1600,
      height: 900,
    }

    // The asset ships a sample route hidden by default — keep it that way.
    el.querySelector('#route-layer')?.setAttribute('opacity', '0')

    const floor = el.querySelector('g[data-floor]')
    const center =
      destination && floor ? drawDestination(floor as SVGGElement, el, destination) : null

    // Measured so the focused window fills the actual map area — on a phone
    // that area is taller than wide, and a plan-shaped window would letterbox.
    const hostRect = host.getBoundingClientRect()
    const containerAspect =
      hostRect.width > 0 && hostRect.height > 0 ? hostRect.width / hostRect.height : undefined
    const view = center && focused ? focusedViewBox(base, center, containerAspect) : base
    el.setAttribute('viewBox', `${view.x} ${view.y} ${view.width} ${view.height}`)

    if (destination) {
      el.setAttribute('aria-label', `ผัง${floorLabel} — จุดหมาย ${destination.name}`)
    }
  }, [svg, destination, floorLabel, focused])

  return (
    <div className="relative h-full w-full overflow-hidden rounded-lg border border-line bg-surface">
      {/* The plan is a repo-controlled static asset, not user input — inlining
          it is what keeps the destination highlight and pin reachable. */}
      <div
        ref={hostRef}
        className="flex h-full w-full items-center justify-center [&>svg]:m-auto"
        dangerouslySetInnerHTML={{ __html: svg }}
      />
      {destination && (
        <button
          type="button"
          onClick={() => setFocused((value) => !value)}
          className="absolute top-2.5 right-2.5 cursor-pointer rounded-full border border-line bg-surface px-3 py-1.5 font-sans text-caption font-bold text-ink"
        >
          {focused ? 'ดูทั้งชั้น' : 'ดูจุดหมาย'}
        </button>
      )}
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
