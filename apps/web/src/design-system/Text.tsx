import { cn } from 'cn'
import type { ReactNode } from 'react'

/*
 * The type scale lives here rather than in the screens. DESIGN.md fixes eight
 * text tokens; a screen that needs a ninth size is a screen that has drifted,
 * so these wrappers stay even though they are one line of utilities each.
 *
 * `className` exists for one legitimate case: the staff register leads with a
 * smaller headline on a phone and the full one on the console, which is a
 * container-query pair, not a new size.
 */

type TextProps = {
  children: ReactNode
  className?: string
}

export function PageTitle({ children, className }: TextProps) {
  return <h1 className={cn('m-0 font-sans text-h1 text-ink', className)}>{children}</h1>
}

export function SectionTitle({ children, className }: TextProps) {
  return <h2 className={cn('m-0 mb-3 font-sans text-h2 text-ink', className)}>{children}</h2>
}

export function Lead({ children, className }: TextProps) {
  return <p className={cn('m-0 font-sans text-body-md text-ink-muted', className)}>{children}</p>
}

export function Meta({ children, className }: TextProps) {
  return <p className={cn('m-0 font-sans text-caption text-ink-muted', className)}>{children}</p>
}

/** Short code or numeral — Space Grotesk, never used for prose. */
export function Code({ children, className }: TextProps) {
  return <span className={cn('font-code text-label-code text-ink', className)}>{children}</span>
}

/** Staff headline: a board on a phone, a page title on the console. */
export const staffTitle = 'text-[20px]/[1.28] font-bold @7xl:text-h1'
/** Staff section heading, same idea one level down. */
export const staffSection = 'text-[14px]/[1.3] font-bold @7xl:text-h2'
