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

/*
 * Station glyphs — added for the login, registration and queue-call screens.
 * CallIcon is the only one that is not a generic mark: it is the floor plan's
 * own vocabulary, a route arriving at a node, because calling a ticket is
 * exactly the moment a patient is sent along that route.
 */

export function SearchIcon(props: IconProps) {
  return (
    <Glyph size={18} {...props}>
      <circle cx="11" cy="11" r="6.5" />
      <line x1="16" y1="16" x2="20.5" y2="20.5" />
    </Glyph>
  )
}

export function PlusIcon(props: IconProps) {
  return (
    <Glyph size={16} strokeWidth={2.5} {...props}>
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </Glyph>
  )
}

export function SkipIcon(props: IconProps) {
  return (
    <Glyph size={16} {...props}>
      <polyline points="7 6 13 12 7 18" />
      <line x1="17" y1="6" x2="17" y2="18" />
    </Glyph>
  )
}

/** A route arriving at a navigation node — "send the next patient here". */
export function CallIcon(props: IconProps) {
  return (
    <Glyph size={18} {...props}>
      <line x1="3" y1="12" x2="11.5" y2="12" />
      <polyline points="8.5 8.5 12 12 8.5 15.5" />
      <circle cx="18" cy="12" r="3" />
    </Glyph>
  )
}

export function SignOutIcon(props: IconProps) {
  return (
    <Glyph size={16} {...props}>
      <path d="M14 4h4a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-4" />
      <line x1="4" y1="12" x2="13" y2="12" />
      <polyline points="9.5 8.5 13 12 9.5 15.5" />
    </Glyph>
  )
}

/** Registration desk: the day's list a clerk opens a visit on. */
export function RegistrationIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <path d="M8 4H6a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1h-2" />
      <rect x="8.5" y="2.5" width="7" height="3.5" rx="1" />
      <line x1="9" y1="11" x2="15" y2="11" />
      <line x1="9" y1="15" x2="13" y2="15" />
    </Glyph>
  )
}

/** Service point: tickets stacked at a counter. */
export function QueueIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <rect x="3" y="5" width="18" height="5" rx="1.5" />
      <rect x="5" y="13" width="14" height="4" rx="1.5" />
      <line x1="8" y1="20" x2="16" y2="20" />
    </Glyph>
  )
}

/** Spoken guidance: a speaker announcing the route (#108, FR-25). */
export function VoiceIcon(props: IconProps) {
  return (
    <Glyph {...props}>
      <path d="M11 5 6.5 9H4a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h2.5L11 19z" />
      <path d="M15 9a4.2 4.2 0 0 1 0 6" />
      <path d="M17.7 6.5a8 8 0 0 1 0 11" />
    </Glyph>
  )
}
