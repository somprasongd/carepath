import { describe, expect, it } from 'vitest'
import { focusedViewBox, parseViewBox } from './FloorPlanMap'

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
