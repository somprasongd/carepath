/**
 * CarePath design tokens.
 *
 * Source of truth is the DESIGN.md frontmatter at the repo root, which itself
 * mirrors the `:root` custom properties in `packages/floorplans/floors/*.svg`
 * (ADR-0003 — the app chrome and the hospital signage are one system).
 *
 * `src/styles/index.css` carries the same values as CSS custom properties and
 * is what the components actually style against. This module exists so
 * TypeScript and SVG code can read a token, and so the /design catalogue can
 * enumerate them. If a value changes, change it in DESIGN.md,
 * src/styles/index.css and here together.
 */

export const colors = {
  primary: '#ff7a00',
  'primary-tint': '#fff1e2',
  secondary: '#0b6e99',
  'secondary-tint': '#e3f1f7',
  ink: '#1f2b33',
  'ink-muted': '#66727d',
  neutral: '#f4f6f5',
  surface: '#ffffff',
  line: '#dde3e3',
  success: '#1f7a5c',
  'success-tint': '#e7f5ef',
  warning: '#9a5310',
  'warning-tint': '#fff1e2',
} as const

export type ColorToken = keyof typeof colors

export type Zone =
  | 'public'
  | 'opd'
  | 'diagnostic'
  | 'pharmacy'
  | 'rehab'
  | 'ipd'
  | 'support'

/**
 * Zone tints are copied byte-for-byte from the floor-plan SVGs. `edge` is the
 * hairline used on the chip dot and on map rectangles so a pale tint still
 * reads as a shape; chip text stays `ink` (every tint is pale — see DESIGN.md).
 */
export const zones: Record<Zone, { label: string; fill: string; edge: string; use: string }> = {
  public: {
    label: 'Public',
    fill: '#fff7df',
    edge: '#d8c98f',
    use: 'ต้อนรับ / การเงิน / จุดรอ',
  },
  opd: {
    label: 'OPD',
    fill: '#dcecff',
    edge: '#a8c6ea',
    use: 'ห้องตรวจผู้ป่วยนอก',
  },
  diagnostic: {
    label: 'Diagnostic',
    fill: '#e7e0ff',
    edge: '#b6a6e0',
    use: 'Lab / X-ray',
  },
  pharmacy: {
    label: 'เภสัชกรรม',
    fill: '#e4f7df',
    edge: '#9fcf8e',
    use: 'ห้องยา',
  },
  rehab: {
    label: 'Rehab',
    fill: '#fff1c9',
    edge: '#e0c477',
    use: 'กายภาพบำบัด',
  },
  ipd: {
    label: 'IPD',
    fill: '#dff5f4',
    edge: '#8fc9c4',
    use: 'หอผู้ป่วยใน',
  },
  support: {
    label: 'Support',
    fill: '#eceff2',
    edge: '#c4cbd0',
    use: 'พื้นที่สนับสนุน',
  },
}

export const typography = {
  'display-stat': { family: 'Space Grotesk', size: '34px', weight: 700, lineHeight: 1.1 },
  h1: { family: 'Noto Sans Thai', size: '26px', weight: 700, lineHeight: 1.28 },
  h2: { family: 'Noto Sans Thai', size: '16px', weight: 700, lineHeight: 1.3 },
  'body-md': { family: 'Noto Sans Thai', size: '14px', weight: 400, lineHeight: 1.6 },
  'body-sm': { family: 'Noto Sans Thai', size: '13px', weight: 400, lineHeight: 1.5 },
  'label-code': { family: 'Space Grotesk', size: '13px', weight: 700, lineHeight: 1 },
  caption: { family: 'Noto Sans Thai', size: '12px', weight: 500, lineHeight: 1.4 },
  button: { family: 'Space Grotesk', size: '16px', weight: 700, lineHeight: 1 },
} as const

export type TypographyToken = keyof typeof typography

export const radius = {
  sm: '8px',
  md: '12px',
  lg: '16px',
  xl: '20px',
  full: '9999px',
  ticket: '4px 16px 16px 16px',
} as const

export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '12px',
  lg: '16px',
  xl: '20px',
  '2xl': '24px',
  '3xl': '32px',
  '4xl': '40px',
} as const

export const layout = {
  gutter: '20px',
  marginDesktop: '40px',
  patientMaxWidth: '480px',
  staffRailWidth: '220px',
  touchTargetMin: '44px',
} as const
