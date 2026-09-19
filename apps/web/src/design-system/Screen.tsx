import type { CSSProperties, ReactNode } from 'react'

export type ScreenVariant = 'patient' | 'staff-mobile' | 'desktop'

/**
 * The frame every reference screen sits in. Patient and staff mobile are a
 * 390×844 LIFF viewport; desktop is the 1440×900 front-desk console.
 */
export function Screen({
  variant,
  children,
}: {
  variant: ScreenVariant
  children: ReactNode
}) {
  return <div className={`cp-screen cp-screen--${variant}`}>{children}</div>
}

/** Scrolling body of a mobile screen; leaves room for the dock underneath. */
export function ScreenScroll({
  children,
  style,
}: {
  children: ReactNode
  style?: CSSProperties
}) {
  return (
    <div className="cp-screen__scroll" style={style}>
      {children}
    </div>
  )
}

/** Scrolling body of the desktop console, beside the side rail. */
export function ScreenMain({ children }: { children: ReactNode }) {
  return <main className="cp-screen__main">{children}</main>
}

/** Pinned to the bottom edge: the sticky action bar, sheet or tab bar. */
export function ScreenDock({ children }: { children: ReactNode }) {
  return <div className="cp-screen__dock">{children}</div>
}
