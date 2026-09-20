import { cn } from 'cn'
import type { ReactNode } from 'react'
import { ChevronLeftIcon } from './Icon'

export type AppBarProps = {
  /** Wordmark line; omit when the screen leads with a back chevron instead. */
  wordmark?: ReactNode
  title?: string
  subtitle?: string
  /** Patient screens have no persistent nav — only this back chevron. */
  onBack?: () => void
  /** Spoken label for the back chevron; pass the translated phrase. */
  backLabel?: string
  trailing?: ReactNode
  className?: string
}

export function AppBar({
  wordmark,
  title,
  subtitle,
  onBack,
  backLabel,
  trailing,
  className
}: AppBarProps) {
  return (
    <header
      className={cn(
        'flex shrink-0 items-center justify-between gap-3 px-gutter pt-[22px] pb-2',
        className,
      )}
    >
      {onBack && (
        <button
          type="button"
          className="flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-[10px] border border-line bg-surface text-ink"
          aria-label={backLabel}
          onClick={onBack}
        >
          <ChevronLeftIcon />
        </button>
      )}
      {wordmark && <div className="font-code text-[18px] font-bold text-ink">{wordmark}</div>}
      {title && (
        <div className="min-w-0 flex-1">
          <div className="font-sans text-h2 text-ink">{title}</div>
          {subtitle && (
            <div className="mt-px font-sans text-caption font-normal text-ink-muted">
              {subtitle}
            </div>
          )}
        </div>
      )}
      {trailing}
    </header>
  )
}
