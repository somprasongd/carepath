import { describe, expect, it } from 'vitest'
import { locationQrPayload, qrModules } from './qr'

describe('locationQrPayload', () => {
  it('carries a bare place reference the QR provider parses', () => {
    expect(locationQrPayload('REG-01')).toBe('location/REG-01')
  })
})

describe('qrModules', () => {
  it('encodes a scannable code: square side, finder-pattern corners dark', () => {
    const { path, moduleCount } = qrModules('location/REG-01')
    // The three finder patterns pin a QR's corners — cheap structural proof
    // the path is a real code, not an empty/garbled matrix.
    expect(moduleCount).toBeGreaterThanOrEqual(21)
    expect(moduleCount % 2).toBe(1)
    expect(path).toContain('M0 0h1v1h-1z') // top-left finder's corner module
    expect(path).toContain(`M${moduleCount - 7} 0h1v1h-1z`) // top-right finder
    expect(path).toContain(`M0 ${moduleCount - 7}h1v1h-1z`) // bottom-left finder
  })

  it('grows with longer payloads', () => {
    const short = qrModules('location/REG-01').moduleCount
    const long = qrModules(
      'location/SOME-VERY-LONG-PLACE-IDENTIFIER-THAT-FORCES-A-BIGGER-TYPE',
    ).moduleCount
    expect(long).toBeGreaterThanOrEqual(short)
  })
})
