import { describe, expect, it } from 'vitest'
import { groupWarnings } from './-FloorPlan'
import type { FloorPlanWarning } from '@/features/floorplan'

describe('groupWarnings', () => {
  it('groups refs by code and labels each group in Thai', () => {
    const groups = groupWarnings([
      { code: 'PLACE_MISSING', ref: 'LAB-01' },
      { code: 'NODE_MISSING', ref: 'node-lab' },
      { code: 'PLACE_MISSING', ref: 'CT-01' },
    ])

    expect(groups).toHaveLength(2)
    const missing = groups.find((g) => g.code === 'PLACE_MISSING')
    expect(missing?.refs).toEqual(['LAB-01', 'CT-01'])
    expect(missing?.label).toBe('จุดที่ผังยังไม่ได้วาด')
  })

  // The contract types the code as a closed enum, so this cast is the only
  // way to express the case it exists for: a client running older generated
  // types against a server that has since added a warning kind. Showing the
  // raw code beats silently dropping the warning — an admin can still act on
  // "SOMETHING_NEW · LAB-01".
  it('falls back to the raw code rather than dropping a warning it cannot name', () => {
    const [group] = groupWarnings([
      { code: 'SOMETHING_NEW', ref: 'X-01' } as unknown as FloorPlanWarning,
    ])
    expect(group.label).toBe('SOMETHING_NEW')
    expect(group.refs).toEqual(['X-01'])
  })

  it('is empty for a clean plan', () => {
    expect(groupWarnings([])).toEqual([])
  })
})
