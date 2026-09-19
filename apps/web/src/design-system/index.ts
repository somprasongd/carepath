/**
 * CarePath design system.
 *
 * Spec: DESIGN.md at the repo root. Visual reference: docs/designs/patient-staff-ui.html.
 * Every component here styles itself from the tokens in src/styles/index.css —
 * if you find yourself writing a hex value in a screen, the token is missing.
 *
 * `ui/` holds files the shadcn CLI generated, edited in place to carry
 * CarePath's variants rather than the stock ones. Everything beside it is
 * ours, because the shadcn equivalent's structure or behaviour did not fit;
 * each file's header says why.
 *
 * Domain compositions (a patient needing attention, a service-point row) live
 * in features/, not here — this module must stay ignorant of the API.
 */

export * from './tokens'
export * from './Icon'
export * from './Text'
export * from './Button'
export * from './ui/badge'
export * from './Card'
export * from './Field'
export * from './ChoiceCard'
export * from './ZoneChip'
export * from './StatusBadge'
export * from './JourneyRail'
export * from './StatCard'
export * from './Note'
export * from './DataTable'
export * from './BottomSheet'
export * from './StickyActionBar'
export * from './Navigation'
export * from './SchematicMap'
export * from './Screen'
