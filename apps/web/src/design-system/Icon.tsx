/**
 * CarePath icon set — drawn in the same stroke grammar as the floor plans
 * (2px round caps, 24px box). DESIGN.md forbids a generic map pin from a
 * stock set; the location mark here is the SVG's own node shape.
 */

export type IconProps = {
  size?: number
  strokeWidth?: number
  className?: string
}

type BaseProps = IconProps & { children: React.ReactNode }

function Glyph({ size = 20, strokeWidth = 2, className, children }: BaseProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      {children}
    </svg>
  )
}

export function CheckIcon(props: IconProps) {
  return (
    <Glyph strokeWidth={3} size={12} {...props}>
      <polyline points="4 12 9 17 20 6" />
    </Glyph>
  )
}

export function ChevronRightIcon(props: IconProps) {
  return (
    <Glyph size={18} strokeWidth={2.5} {...props}>
      <polyline points="9 6 15 12 9 18" />
    </Glyph>
  )
}

export function ChevronLeftIcon(props: IconProps) {
  return (
    <Glyph size={16} strokeWidth={2.5} {...props}>
      <polyline points="15 6 9 12 15 18" />
    </Glyph>
  )
}

export function LocationIcon(props: IconProps) {
  return (
    <Glyph size={16} {...props}>
      <path d="M12 21s-7-6.1-7-11a7 7 0 0 1 14 0c0 4.9-7 11-7 11z" />
      <circle cx="12" cy="10" r="2.4" />
    </Glyph>
  )
}

export function InfoIcon(props: IconProps) {
  return (
    <Glyph size={16} {...props}>
      <circle cx="12" cy="12" r="9" />
      <line x1="12" y1="8" x2="12" y2="12.5" />
      <line x1="12" y1="16" x2="12" y2="16.01" />
    </Glyph>
  )
}

export function OverviewIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <path d="M3 11l9-7 9 7" />
      <path d="M5 10v9a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1v-9" />
    </Glyph>
  )
}

export function ServicePointsIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <rect x="3" y="3" width="8" height="8" rx="1.5" />
      <rect x="13" y="3" width="8" height="8" rx="1.5" />
      <rect x="3" y="13" width="8" height="8" rx="1.5" />
      <rect x="13" y="13" width="8" height="8" rx="1.5" />
    </Glyph>
  )
}

export function FloorPlanIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <rect x="4" y="3" width="16" height="18" rx="1.5" />
      <line x1="8" y1="8" x2="8" y2="8.01" />
      <line x1="12" y1="8" x2="12" y2="8.01" />
      <line x1="16" y1="8" x2="16" y2="8.01" />
      <line x1="8" y1="13" x2="8" y2="13.01" />
      <line x1="12" y1="13" x2="12" y2="13.01" />
      <line x1="16" y1="13" x2="16" y2="13.01" />
    </Glyph>
  )
}

export function PatientsIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <circle cx="9" cy="9" r="3" />
      <circle cx="16.5" cy="10.5" r="2.4" />
      <path d="M3.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5" />
      <path d="M14.5 15.2c2.3.3 3.9 2 3.9 3.8" />
    </Glyph>
  )
}
