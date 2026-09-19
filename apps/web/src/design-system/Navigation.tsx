import { cn } from 'cn'
import type { ReactNode } from 'react'
import { ChevronLeftIcon } from './Icon'

export type NavItem = {
  id: string
  label: string
  icon: ReactNode
}

export type NavProps = {
  items: NavItem[]
  activeId: string
  onSelect?: (id: string) => void
  className?: string
}

/** Staff mobile: 4 destinations, thumb height, active item in node blue. */
export function BottomTabBar({ items, activeId, onSelect, className }: NavProps) {
  return (
    <nav
      className={cn('flex h-[72px] border-t border-line bg-surface', className)}
      aria-label="เมนูเจ้าหน้าที่"
    >
      {items.map((item) => (
        <button
          type="button"
          key={item.id}
          className={cn(
            'flex flex-1 cursor-pointer flex-col items-center justify-center gap-1 border-0 bg-none font-sans text-[10px] no-underline',
            item.id === activeId ? 'font-bold text-secondary' : 'font-semibold text-ink-muted'
          )}
          aria-current={item.id === activeId ? 'page' : undefined}
          onClick={() => onSelect?.(item.id)}
        >
          {item.icon}
          {item.label}
        </button>
      ))}
    </nav>
  )
}

export type SideRailProps = NavProps & {
  /** Who is signed in, e.g. "ประชาสัมพันธ์". */
  role: string
  /** Shown as a sign-out button under the role card when the session is real. */
  onSignOut?: () => void
}

/** Staff desktop: the same 4 destinations as a fixed 220px rail. */
export function SideRail({ items, activeId, onSelect, role, onSignOut, className }: SideRailProps) {
  return (
    <nav
      className={cn(
        'flex w-[220px] shrink-0 flex-col border-r border-line bg-surface px-[18px] py-6',
        className
      )}
      aria-label="เมนูคอนโซลเจ้าหน้าที่"
    >
      <div className="mb-8 px-1.5">
        <div className="font-code text-[18px] font-bold text-ink">CarePath</div>
        <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">
          คอนโซลเจ้าหน้าที่
        </div>
      </div>

      {items.map((item) => (
        <button
          type="button"
          key={item.id}
          className={cn(
            'mb-1 flex cursor-pointer items-center gap-2.5 rounded-[10px] border-0 px-3 py-2.5 text-left font-sans text-body-sm no-underline',
            item.id === activeId
              ? 'bg-secondary-tint font-bold text-secondary'
              : 'bg-none font-medium text-ink-muted'
          )}
          aria-current={item.id === activeId ? 'page' : undefined}
          onClick={() => onSelect?.(item.id)}
        >
          {item.icon}
          {item.label}
        </button>
      ))}

      <div className="flex-1" />

      <div className="rounded-[10px] bg-neutral px-3 py-2.5">
        <div className="mb-0.5 text-[10px] text-ink-muted">เข้าสู่ระบบในฐานะ</div>
        <div className="font-sans text-body-sm font-bold text-ink">{role}</div>
        {onSignOut && (
          <button
            type="button"
            className="mt-2 cursor-pointer border-0 bg-none p-0 font-sans text-caption font-semibold text-ink-muted underline underline-offset-2 hover:text-ink"
            onClick={onSignOut}
          >
            ออกจากระบบ
          </button>
        )}
      </div>
    </nav>
  )
}

export type AppBarProps = {
  /** Wordmark line; omit when the screen leads with a back chevron instead. */
  wordmark?: ReactNode
  title?: string
  subtitle?: string
  /** Patient screens have no persistent nav — only this back chevron. */
  onBack?: () => void
  backLabel?: string
  trailing?: ReactNode
  className?: string
}

export function AppBar({
  wordmark,
  title,
  subtitle,
  onBack,
  backLabel = 'ย้อนกลับ',
  trailing,
  className,
}: AppBarProps) {
  return (
    <header
      className={cn(
        'flex shrink-0 items-center justify-between gap-3 px-gutter pt-[22px] pb-2',
        className
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

/** The quiet second half of the staff wordmark: "CarePath · เจ้าหน้าที่". */
export function WordmarkSub({ children }: { children: ReactNode }) {
  return <span className="font-medium text-ink-muted">{children}</span>
}
