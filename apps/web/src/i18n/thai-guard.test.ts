import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

/**
 * The #93 gate: no hardcoded Thai left in patient-scope files. Every
 * patient-facing string goes through the catalogs (ADR-0012); a Thai
 * character outside `locales/th.ts` in this scope is a regression.
 *
 * This is a Node (fs) test on purpose: macOS grep has no -P and ripgrep
 * chokes on Thai unicode ranges, and both silently return "clean" — only a
 * JS regex over the file bytes is trustworthy here (issue #93 warning).
 *
 * Scope: patient screens, their feature modules, shared design-system
 * components, and i18n itself. Out of scope with reasons — the list should
 * only ever shrink:
 * - Staff console surfaces pin locale 'th' by design (ADR-0012 §2): the
 *   staff features inside features/visit, and the staff chrome inside
 *   design-system (Navigation, StatusBadge, tokens).
 * - th.ts is the catalog itself; *.test.* files are covered by the
 *   English-render test instead.
 */
const WEB_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..')

const SCOPE_ROOTS = [
  'src/routes/patient',
  'src/routes/-SharedScreen.tsx',
  'src/auth',
  'src/features/visit',
  'src/features/navigation',
  'src/features/floorplan',
  'src/features/share',
  'src/i18n',
  'src/design-system',
]

const EXCLUDED = new Set([
  // ADR-0012 §2 — staff console surfaces, pinned Thai by design.
  'src/features/visit/staff.ts',
  'src/features/visit/components/StaffStatusBadge.tsx',
  'src/features/visit/components/StaffVisitDetail.tsx',
  'src/features/visit/components/StaffVisitList.tsx',
  'src/features/visit/components/StepTransitionControls.tsx',
  'src/design-system/Navigation.tsx',
  'src/design-system/StatusBadge.tsx',
  'src/design-system/tokens.ts',
  // The Thai catalog itself.
  'src/i18n/locales/th.ts',
])

const THAI = /[\u0E00-\u0E7F]/

function walk(root: string): string[] {
  const entries = readdirSync(root)
  const files: string[] = []
  for (const entry of entries) {
    const full = join(root, entry)
    if (statSync(full).isDirectory()) {
      files.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry) && !/\.test\./.test(entry)) {
      files.push(full)
    }
  }
  return files
}

function scopedFiles(): string[] {
  const files: string[] = []
  for (const root of SCOPE_ROOTS) {
    const full = join(WEB_ROOT, root)
    if (statSync(full).isDirectory()) {
      files.push(...walk(full))
    } else {
      files.push(full)
    }
  }
  return files.filter((file) => !EXCLUDED.has(file.slice(WEB_ROOT.length + 1)))
}

describe('patient-scope Thai guard (#93)', () => {
  it('scans a non-empty file set — the walker itself is not silently broken', () => {
    expect(scopedFiles().length).toBeGreaterThan(20)
  })

  it('keeps every scanned file free of hardcoded Thai', () => {
    const offenders = scopedFiles().filter((file) => THAI.test(readFileSync(file, 'utf8')))
    const relative = offenders.map((file) => file.slice(WEB_ROOT.length + 1))
    expect(
      relative,
      'Hardcoded Thai found in patient scope — move it to locales/th.ts ' +
        '(or, for a staff-only surface, add it to EXCLUDED with an ADR-0012 §2 reason)',
    ).toEqual([])
  })
})
