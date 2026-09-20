import { describe, expect, it } from 'vitest'
import { clampView, focusedViewBox, MAX_ZOOM, parseViewBox, routedViewBox, zoomedView } from './FloorPlanMap'

describe('parseViewBox', () => {
  it('parses a standard four-value viewBox', () => {
    expect(parseViewBox('0 0 1600 900')).toEqual({ x: 0, y: 0, width: 1600, height: 900 })
    expect(parseViewBox('70 130 640\t360')).toEqual({ x: 70, y: 130, width: 640, height: 360 })
  })

  it('returns null for absent or malformed values', () => {
    expect(parseViewBox(null)).toBeNull()
    expect(parseViewBox('')).toBeNull()
    expect(parseViewBox('0 0')).toBeNull()
    expect(parseViewBox('a b c d')).toBeNull()
    expect(parseViewBox('0 0 -10 900')).toBeNull()
    expect(parseViewBox('0 0 1600 0')).toBeNull()
  })
})

describe('focusedViewBox', () => {
  const base = { x: 0, y: 0, width: 1600, height: 900 }

  it('centres the focus window on a mid-plan destination', () => {
    expect(focusedViewBox(base, { x: 800, y: 450 })).toEqual({
      x: 480,
      y: 270,
      width: 640,
      height: 360,
    })
  })

  it('clamps near the edges so the focus never leaves the floor', () => {
    expect(focusedViewBox(base, { x: 0, y: 0 })).toEqual({ x: 0, y: 0, width: 640, height: 360 })
    expect(focusedViewBox(base, { x: 1600, y: 900 })).toEqual({
      x: 960,
      y: 540,
      width: 640,
      height: 360,
    })
  })

  it('returns the whole plan when the plan is smaller than the focus window', () => {
    const small = { x: 0, y: 0, width: 500, height: 300 }
    expect(focusedViewBox(small, { x: 250, y: 150 })).toEqual(small)
  })

  it('matches the container aspect so a tall phone map fills edge to edge', () => {
    // Square container: 640 wide at aspect 1 fits the 900-unit plan height.
    expect(focusedViewBox(base, { x: 800, y: 450 }, 1)).toEqual({
      x: 480,
      y: 130,
      width: 640,
      height: 640,
    })
    // Very tall container: capped by the plan height, so the window narrows.
    expect(focusedViewBox(base, { x: 800, y: 450 }, 0.5)).toEqual({
      x: 575,
      y: 0,
      width: 450,
      height: 900,
    })
    // Wide container: the window shortens instead of growing past 640 wide.
    expect(focusedViewBox(base, { x: 800, y: 450 }, 4)).toEqual({
      x: 480,
      y: 370,
      width: 640,
      height: 160,
    })
  })

  it('falls back to the plan shape for an unusable aspect', () => {
    expect(focusedViewBox(base, { x: 800, y: 450 }, Number.NaN)).toEqual(
      focusedViewBox(base, { x: 800, y: 450 }),
    )
    expect(focusedViewBox(base, { x: 800, y: 450 }, 0)).toEqual(
      focusedViewBox(base, { x: 800, y: 450 }),
    )
  })
})

describe('routedViewBox', () => {
  const base = { x: 0, y: 0, width: 1600, height: 900 }

  it('centres a short walk like the destination focus does (never narrower than 640)', () => {
    expect(
      routedViewBox(base, { from: { x: 700, y: 200 }, to: { x: 900, y: 300 } }),
    ).toEqual({ x: 480, y: 70, width: 640, height: 360 })
  })

  it('widens past 640 so a long line stays whole on screen', () => {
    // A 1000-unit walk plus padding exceeds the 640 minimum; the window
    // grows to 1180 and clamps to the floor's top edge.
    expect(routedViewBox(base, { from: { x: 100, y: 200 }, to: { x: 1100, y: 300 } })).toEqual({
      x: 10,
      y: 0,
      width: 1180,
      height: 663.75,
    })
  })

  it('never shows space outside the floor', () => {
    const view = routedViewBox(base, { from: { x: 0, y: 0 }, to: { x: 1600, y: 900 } })
    expect(view).toEqual(base)
  })

  it('matches the container aspect so a tall phone fills edge to edge', () => {
    expect(routedViewBox(base, { from: { x: 700, y: 200 }, to: { x: 900, y: 300 } }, 1)).toEqual({
      x: 480,
      y: 0,
      width: 640,
      height: 640,
    })
  })

  it('keeps the 640-unit readability width on a tall phone map area', () => {
    // The navigate screen's map area is taller than wide (~348×756). Filling
    // that shape would need a window 1391 units tall — past the floor — so
    // the height clamps to the floor and the width must stay at the
    // readability floor, not shrink to the container's narrow shape and clip
    // the line off-screen (the svg letterboxes inside the area instead).
    expect(
      routedViewBox(base, { from: { x: 254, y: 307 }, to: { x: 531, y: 473 } }, 348 / 756),
    ).toEqual({ x: 72.5, y: 0, width: 640, height: 900 })
  })
})

describe('clampView', () => {
  const base = { x: 0, y: 0, width: 1600, height: 900 }

  it('pulls an out-of-bounds window back inside the plan (zoomed-in axes stay edge-locked)', () => {
    expect(
      clampView({ x: -100, y: 700, width: 640, height: 640 }, base),
    ).toEqual({ x: 0, y: 260, width: 640, height: 640 })
  })

  it('shrinks a window larger than the plan to the whole plan', () => {
    const clamped = clampView({ x: 300, y: 100, width: 2000, height: 1125 }, base)
    expect({ width: clamped.width, height: clamped.height }).toEqual({ width: 1600, height: 900 })
  })

  it('lets a fully-out axis drift so the plan can slide out from under the bottom sheet', () => {
    // The plan exactly fills the window vertically (900 == 900): dragging up
    // shifts the view above the plan, but never past 75% of the window.
    const drifted = clampView({ x: 0, y: -400, width: 1600, height: 900 }, base)
    expect(drifted.y).toBe(-400)
    const tooFar = clampView({ x: 0, y: -800, width: 1600, height: 900 }, base)
    expect(tooFar.y).toBe(-675) // keeps 25% of the window on the plan
  })
})

describe('zoomedView', () => {
  const base = { x: 0, y: 0, width: 1600, height: 900 }
  const view = { x: 480, y: 270, width: 640, height: 360 }

  it('zooms in at the cursor: the point under the cursor stays put', () => {
    const next = zoomedView(view, base, 0.5, 0.25, 0.75)
    expect(next.width).toBe(320)
    // The plan point at viewport (0.25, 0.75): (480 + 0.25*640, 270 + 0.75*360)
    const anchorX = 480 + 0.25 * 640
    const anchorY = 270 + 0.75 * 360
    expect(next.x + 0.25 * next.width).toBeCloseTo(anchorX)
    expect(next.y + 0.75 * next.height).toBeCloseTo(anchorY)
  })

  it('never zooms out past the whole plan', () => {
    expect(zoomedView(view, base, 10)).toEqual(base)
  })

  it('never zooms in past MAX_ZOOM of the plan width', () => {
    const maxedIn = zoomedView(view, base, 1 / 1_000_000)
    expect(maxedIn.width).toBeCloseTo(1600 / MAX_ZOOM)
  })
})
