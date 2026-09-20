import { cn } from 'cn'
import type { ReactNode } from 'react'

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

/** The quiet second half of the staff wordmark: "CarePath · <role>". */
export function WordmarkSub({ children }: { children: ReactNode }) {
  return <span className="font-medium text-ink-muted">{children}</span>
}
