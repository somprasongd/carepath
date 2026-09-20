# ADR-0012: The Client Owns Display Text; the Server Sends Stable Codes

- Status: Accepted
- Date: 2026-09-20
- Relates to: FR-19 (#91), [ADR-0009](0009-carepath-owns-journey-plan.md) (step identity by `kind`/`clinicCode`), [ADR-0006](0006-rest-openapi.md) (contract as source of truth)

## Context

FR-19 (Should · S4): foreign patients must be able to switch the patient app
to English at runtime, across the patient screens and the navigation
instructions (US-11). Today nothing supports this — no i18n mechanism exists
in `apps/web`, Thai UI strings are hardcoded in components, step titles come
from a `thaiStepTitle()` map, and time formatting hardcodes `'th-TH'`.

The strings in code are not the hard part. The hard part is **who owns the
words the patient reads when the data behind them is single-language**:

- `carepath.service_point.name` is one column and its seeds are already mixed:
  `Registration`, `Laboratory`, `X-Ray`, `OPD`, `Cashier` (English) next to
  `อายุรกรรม (MED)` (Thai) — migrations 000002, 000004, 000007, 000010, 000013.
  `stepMeta()` renders that raw `name` to patients today.
- Mock HIS returns Thai clinic names inside visit facts.

Pure string extraction would therefore leave service point names untranslated
in English mode — and inconsistently bilingual in Thai mode. This is an
architecture decision, not a refactor.

## Decision

### 1. The server keeps sending stable codes; the web app maps codes to display text per locale

The API stays locale-ignorant. Display text for anything addressed by a code
lives in client-side catalogs keyed by locale:

| Text | Key | Already-stable code |
| --- | --- | --- |
| Step titles | `kind`, plus `clinicCode` for CLINIC steps | ADR-0009 step identity (`stepKey` parts) |
| Service point names | `ServicePoint.code` | required in the contract: `REGISTRATION`, `LAB`, `XRAY`, `CASHIER`, `PHARMACY`, `CLINIC:MED`, `ORDERTYPE:*`, `DOCTOR` |

**No contract or DB change is needed.** `ServicePoint.code` is already the
required, unique binding key (`GET /api/v1/service-points/{code}`), and
`thaiStepTitle()` already demonstrated this lookup pattern for step kinds.

**Fallback rule:** an unmapped code degrades to a readable screen, never a
crash or blank — the service point falls back to the server's `name`, a step
kind falls back to the raw kind, exactly as `thaiStepTitle` did before this
ADR.

**Rejected alternative — server-translated text per `Accept-Language`:** it
ties the API to the language set, requires an i18n table plus CRUD from day
one, and over-serves a two-language MVP. It becomes the right answer when the
service point set opens up (§4).

### 2. What is explicitly *not* UI text

- **Patient names are proper nouns** — stored and displayed verbatim, never
  translated, in any locale.
- **The DB `name` column stays as-is.** It remains the staff-facing label
  (staff screens keep reading it) and the patient-facing fallback for unmapped
  codes. No migration; new seeds may keep using either language.
- **Staff console vocabulary is out of scope.** FR-19 is a patient
  requirement; staff are assumed Thai-working. Staff code paths pin the
  locale to `'th'` explicitly rather than accidentally inheriting a patient
  toggle later.

### 3. Mechanism: typed catalogs + React context, no i18n framework

- `apps/web/src/i18n/locales/th.ts` is the canonical catalog; every other
  locale is typed as the same shape (`en.ts: Catalog`). **A missing key in
  `en.ts` is a compile error** — a forgotten translation breaks
  `npm run build`, it never leaks Thai onto an English screen.
- `LocaleProvider` / `useLocale()` / `useT()` provide the locale and a
  type-safe `t(key, params)` with simple `{placeholder}` interpolation. No
  plural rules yet; none of the current text needs them.
- Chosen over `react-i18next` deliberately: the patient-facing surface is
  ~2,000 characters across two locales; type-safe keys come from TypeScript
  directly, matching the repo convention that types are the contract; and the
  root AGENTS.md rule — small working increments over speculative
  architecture — argues against a dependency nothing here needs yet.

### 4. Migration triggers — when this decision must be revisited

- **FR-11 (#105) lands** (admin creates/edits service points): codes become
  an open set and admin-set names cannot live in a client catalog. At that
  point add the server-side i18n table and `Accept-Language` negotiation for
  service point names (the rejected alternative above), keeping the client
  catalog for closed code sets (step kinds) if it still pays for itself.
- **A third language, lazy per-locale loading, plurals/ICU, or translator
  tooling is needed:** move the catalogs to `react-i18next`. The `useT()` seam
  keeps every call site stable through that move.

## Consequences

### Positive

- English support costs catalogs only — no backend, contract, or DB work.
- Adding a language is adding one typed file; incomplete work fails the build.
- Thai patients gain consistent Thai service point names on first landing
  (the current mix of English DB names and Thai labels is part of what this
  fixes).

### Trade-offs

- Every new patient-facing code (a step kind, a service point binding) must
  also add catalog entries in both locales, or it renders via fallback.
- Admin-set service point names will not translate until §4's first trigger
  fires — accepted because FR-11 explicitly excludes a CRUD UI for the
  hackathon scope.
- Catalog changes are code changes: rewording a patient-facing string ships
  through a PR, not a content tool.
