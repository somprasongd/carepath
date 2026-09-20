import { describe, expect, it } from 'vitest'
import { positionLabel, waitedLabel } from './labels'

/** #102: the station console's labels — honest blanks, never guessed numbers. */
describe('waitedLabel', () => {
  const now = Date.parse('2026-09-20T04:00:00Z')

  it('counts whole minutes since the step became READY', () => {
    expect(waitedLabel('2026-09-20T03:48:00Z', now)).toBe('รอ 12 นาที')
  })

  it('floors under a minute instead of rounding up to one', () => {
    expect(waitedLabel('2026-09-20T03:59:30Z', now)).toBe('รอไม่ถึงนาที')
  })

  it('renders an honest blank when the timeline has no arrival fact', () => {
    expect(waitedLabel(null, now)).toBe('—')
    expect(waitedLabel(undefined, now)).toBe('—')
  })
})

describe('positionLabel', () => {
  it('places the step in its visit plan, 1-based', () => {
    expect(positionLabel(3, 5)).toBe('ขั้นที่ 3 จาก 5')
  })
})
