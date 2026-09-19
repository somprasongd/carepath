import { cn } from 'cn'
import type { ReactNode } from 'react'

export type ScreenVariant = 'patient' | 'staff'

/*
 * A screen fills whatever box it is given — the viewport at a real route, the
 * fixed mockup size inside the /design catalogue's ScreenFrame. The container
 * owns the dimensions; the screen never states its own.
 *
 * The mobile → desktop switch is a CONTAINER query (`@7xl`, 1280px, the
 * breakpoint DESIGN.md names), not a viewport one. A viewport query would make
 * the catalogue's 390px phone frame render the desktop console, because the
 * browser window behind it is wide. Container queries measure the frame.
 */
export function Screen({
  variant,
  children,
}: {
  variant: ScreenVariant
  children: ReactNode
}) {
  return (
    <div className="@container h-full w-full">
      <div
        className={cn(
          'relative mx-auto flex h-full w-full flex-col overflow-hidden bg-neutral',
          variant === 'patient'
            ? /* LIFF is always a phone — never widen, even on a desktop browser. */
              'max-w-[430px]'
            : 'max-w-[430px] @7xl:max-w-none @7xl:flex-row'
        )}
      >
        {children}
      </div>
    </div>
  )
}

/**
 * The scrolling body. On a phone it leaves room for the dock underneath; at
 * the console breakpoint it takes the desktop gutter and drops the dock space.
 */
export function ScreenBody({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <main
      className={cn(
        'flex-1 overflow-y-auto px-gutter pt-2 pb-24 @7xl:px-gutter-desktop @7xl:py-8',
        className
      )}
    >
      {children}
    </main>
  )
}

/** Pinned to the bottom edge: the sticky action bar, sheet or tab bar. */
export function ScreenDock({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return <div className={cn('absolute right-0 bottom-0 left-0', className)}>{children}</div>
}

/** Vertical rhythm for a list of cards — 12px, or 20px when `lg`. */
export function Stack({
  children,
  lg = false,
  className,
}: {
  children: ReactNode
  lg?: boolean
  className?: string
}) {
  return <div className={cn('flex flex-col', lg ? 'gap-5' : 'gap-3', className)}>{children}</div>
}
