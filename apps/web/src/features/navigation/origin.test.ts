import { describe, expect, it } from 'vitest'
import type { Journey, JourneyStep } from '@/features/visit'
import { assumedOrigin, assumedOriginLabel } from './origin'

function step(over: Partial<JourneyStep> & { stepKey: string }): JourneyStep {
  return {
    sequence: 0,
    kind: 'REGISTRATION',
    status: 'PENDING',
    servicePointId: null,
    ...over,
  }
}

function journey(steps: JourneyStep[]): Journey {
  return {
    visitId: 'VISIT-TEST',
    patientRef: 'PATIENT-TEST',
    status: 'ACTIVE',
    completed: false,
    syncedAt: '2026-09-20T00:00:00Z',
    steps,
    actionable: [],
    recommended: null,
  }
}

const REG_PLACE = {
  id: 'REG-01',
  floorId: 'I-1301',
  name: 'Reception',
  type: 'COUNTER' as const,
  x: 150,
  y: 190,
  entryNodeId: 'I-1301/node-reception',
  floor: { id: 'I-1301', buildingId: 'BLD-I13', code: '1', name: 'Ground Floor', levelOrder: 1 },
}

describe('assumedOrigin', () => {
  it('anchors on the latest completed step with a service point entry node', () => {
    const withPlace = journey([
      step({
        stepKey: 'REGISTRATION',
        status: 'COMPLETED',
        kind: 'REGISTRATION',
        servicePointId: 'SP-REG',
        servicePoint: {
          id: 'SP-REG',
          code: 'REGISTRATION',
          name: 'Registration',
          placeId: 'REG-01',
          place: REG_PLACE,
        },
      }),
      step({ stepKey: 'LAB', status: 'READY', kind: 'LAB' }),
    ])
    expect(assumedOrigin(withPlace, 'th')).toEqual({
      nodeId: 'I-1301/node-reception',
      stepTitle: 'ลงทะเบียน',
    })
    expect(assumedOrigin(withPlace, 'en')?.stepTitle).not.toBe('ลงทะเบียน')
  })

  it('skips completed steps without an entry node and falls back to an earlier one', () => {
    const j = journey([
      step({
        stepKey: 'REGISTRATION',
        status: 'COMPLETED',
        kind: 'REGISTRATION',
        servicePointId: 'SP-REG',
        servicePoint: {
          id: 'SP-REG',
          code: 'REGISTRATION',
          name: 'Registration',
          placeId: 'REG-01',
          place: REG_PLACE,
        },
      }),
      // Completed but unmapped: no service point, must be skipped over.
      step({ stepKey: 'LAB', status: 'COMPLETED', kind: 'LAB' }),
    ])
    expect(assumedOrigin(j, 'th')?.nodeId).toBe('I-1301/node-reception')
  })

  it('returns null when nothing completed resolves to a node', () => {
    expect(assumedOrigin(journey([step({ stepKey: 'LAB', status: 'READY' })]), 'th')).toBeNull()
    expect(assumedOrigin(undefined, 'th')).toBeNull()
  })
})

describe('assumedOriginLabel', () => {
  it('states the assumption, never a claimed fix', () => {
    expect(
      assumedOriginLabel({ nodeId: 'I-1301/node-reception', stepTitle: 'ลงทะเบียน' }, 'th'),
    ).toBe('ตำแหน่งโดยประมาณ · หลังขั้นตอนลงทะเบียน')
  })
})
