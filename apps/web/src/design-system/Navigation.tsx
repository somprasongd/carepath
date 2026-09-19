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
}

/** Staff mobile: 4 destinations, thumb height, active item in node blue. */
export function BottomTabBar({ items, activeId, onSelect }: NavProps) {
  return (
    <nav className="cp-tabs" aria-label="เมนูเจ้าหน้าที่">
      {items.map((item) => (
        <button
          type="button"
          key={item.id}
          className={`cp-tabs__item ${item.id === activeId ? 'is-active' : ''}`}
          aria-current={item.id === activeId ? 'page' : undefined}
          onClick={() => onSelect?.(item.id)}
        >
          {item.icon}
          <span>{item.label}</span>
        </button>
      ))}
    </nav>
  )
}

export type SideRailProps = NavProps & {
  /** Who is signed in — a role, not a person, at a shared front desk. */
  role: string
}

/** Staff desktop: the same 4 destinations as a fixed 220px rail. */
export function SideRail({ items, activeId, onSelect, role }: SideRailProps) {
  return (
    <nav className="cp-siderail" aria-label="เมนูคอนโซลเจ้าหน้าที่">
      <div className="cp-siderail__brand">
        <div className="cp-siderail__wordmark">CarePath</div>
        <div className="cp-siderail__role">คอนโซลเจ้าหน้าที่</div>
      </div>
      {items.map((item) => (
        <button
          type="button"
          key={item.id}
          className={`cp-siderail__item ${item.id === activeId ? 'is-active' : ''}`}
          aria-current={item.id === activeId ? 'page' : undefined}
          onClick={() => onSelect?.(item.id)}
        >
          {item.icon}
          {item.label}
        </button>
      ))}
      <div className="cp-siderail__spacer" />
      <div className="cp-siderail__user">
        <div className="cp-siderail__user-label">เข้าสู่ระบบในฐานะ</div>
        <div className="cp-siderail__user-name">{role}</div>
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
}

export function AppBar({
  wordmark,
  title,
  subtitle,
  onBack,
  backLabel = 'ย้อนกลับ',
  trailing,
}: AppBarProps) {
  return (
    <header className="cp-appbar">
      {onBack && (
        <button type="button" className="cp-backbtn" aria-label={backLabel} onClick={onBack}>
          <ChevronLeftIcon />
        </button>
      )}
      {wordmark && <div className="cp-appbar__wordmark">{wordmark}</div>}
      {title && (
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="cp-appbar__title">{title}</div>
          {subtitle && <div className="cp-appbar__subtitle">{subtitle}</div>}
        </div>
      )}
      {trailing}
    </header>
  )
}
