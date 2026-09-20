import { describe, expect, it } from 'vitest'
import type { ServicePoint } from '@/features/servicepoint'
import { activeFloorFor, pickablePlaces, type PickablePlace } from './-scan-pick'

function sp(
  code: string,
  place: { id: string; name: string; entryNodeId: string; floorCode: string } | null,
): ServicePoint {
  return {
    id: 'SP-' + code,
    code,
    name: code,
    placeId: place?.id ?? 'PLACE-UNMAPPED',
    place: place
      ? {
          id: place.id,
          floorId: 'F-' + place.floorCode,
          name: place.name,
          type: 'ROOM',
          entryNodeId: place.entryNodeId,
          floor: { id: 'F-' + place.floorCode, buildingId: 'BLD', code: place.floorCode, name: 'Floor ' + place.floorCode, levelOrder: 1 },
        }
      : null,
  }
}

describe('pickablePlaces', () => {
  it('dedupes to one entry per place node and keeps first-seen order', () => {
    const places = pickablePlaces([
      sp('CLINIC:MED', { id: 'OPD-NS', name: 'OPD Nurse Station', entryNodeId: 'node-opd-ns', floorCode: '1' }),
      sp('CLINIC:ENT', { id: 'OPD-NS', name: 'OPD Nurse Station', entryNodeId: 'node-opd-ns', floorCode: '1' }),
      sp('LAB', { id: 'LAB-01', name: 'ห้องเจาะเลือด', entryNodeId: 'node-lab', floorCode: '2' }),
    ])
    expect(places).toEqual<PickablePlace[]>([
      { entryNodeId: 'node-opd-ns', name: 'OPD Nurse Station', floorCode: '1' },
      { entryNodeId: 'node-lab', name: 'ห้องเจาะเลือด', floorCode: '2' },
    ])
  })

  it('drops service points without a place or entry node', () => {
    expect(pickablePlaces([sp('CLINIC:OPHTH', null), sp('XRAY', { id: 'X-1', name: 'X', entryNodeId: '', floorCode: '1' })])).toEqual([])
  })

  it('treats an unset query like no places', () => {
    expect(pickablePlaces(undefined)).toEqual([])
  })
})

describe('activeFloorFor', () => {
  it('recovers to the first floor once floors arrive after mount (no latch)', () => {
    // The regression: the query resolved after the list mounted, so the
    // captured initial state was '' while the chips showed real floors.
    expect(activeFloorFor(null, undefined, [])).toBe('')
    expect(activeFloorFor(null, undefined, ['1', '2'])).toBe('1')
  })

  it('prefers the patient pick while it is still a known floor', () => {
    expect(activeFloorFor('2', '1', ['1', '2'])).toBe('2')
  })

  it('falls back when the picked floor disappears from the data', () => {
    expect(activeFloorFor('3', '1', ['1', '2'])).toBe('1')
  })

  it('ignores a default the data no longer has', () => {
    expect(activeFloorFor(null, '9', ['1', '2'])).toBe('1')
  })
})
