import { describe, expect, it } from 'vitest'
import { en } from './locales/en'
import { th } from './locales/th'
import { lookup, messagesFor } from '.'

describe('locale catalogs', () => {
  it('keeps every locale on the Thai catalog’s exact key set', () => {
    // The compile-time guarantee (`en: Catalog`) is the real gate; this test
    // puts the failure in `npm run test` too, where it is seen first.
    expect(Object.keys(en).sort()).toEqual(Object.keys(th).sort())
  })

  it('maps codes with a colon in them (ORDERTYPE:*, CLINIC:*) in both locales', () => {
    expect(th['sp.ORDERTYPE:LAB']).toBeDefined()
    expect(en['sp.ORDERTYPE:LAB']).toBeDefined()
    expect(th['sp.CLINIC:MED']).toBeDefined()
    expect(en['sp.CLINIC:MED']).toBeDefined()
  })
})

describe('messagesFor', () => {
  it('returns the catalog for each supported locale', () => {
    expect(messagesFor('th')).toBe(th)
    expect(messagesFor('en')).toBe(en)
  })
})

describe('lookup', () => {
  it('finds a key built at runtime', () => {
    expect(lookup(th, `step.title.${'LAB'}`)).toBe('เจาะเลือด')
    expect(lookup(en, `sp.${'CLINIC:MED'}`)).toBe('Internal Medicine')
  })

  it('returns undefined for an unknown key so callers can fall back', () => {
    expect(lookup(th, 'sp.NURSERY')).toBeUndefined()
    expect(lookup(en, 'step.title.PHLEBOTOMY')).toBeUndefined()
  })
})
