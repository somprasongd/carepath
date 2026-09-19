import type { ReactNode } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useStaffAuth } from '@/auth/StaffAuthContext'

/**
 * Header for a single-task staff screen (registration, the queue console) —
 * a dedicated workstation view, not part of the four-tab staff console, so it
 * carries no side rail or tab bar. Only "ออกจากระบบ" (revoke the session,
 * ADR-0010 §9) and whatever identifies the task belong here.
 *
 * Wraps onto two rows below the console width rather than switching layout at
 * a named breakpoint — there's no useful intermediate state to design for
 * here, just "does it fit on one line or not".
 */
export function TaskHeader({ role, station }: { role: string; station?: ReactNode }) {
  const navigate = useNavigate()
  const { logout } = useStaffAuth()

  return (
    <header className="flex shrink-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-line bg-surface px-gutter py-3.5 @7xl:px-gutter-desktop @7xl:py-4">
      <div className="flex min-w-0 flex-wrap items-center gap-3.5">
        <span className="font-code text-[16px] font-bold text-ink">CarePath</span>
        {station}
      </div>
      <div className="flex shrink-0 items-center gap-4">
        <span className="rounded-full bg-neutral px-3.5 py-1.5 font-sans text-caption font-normal text-ink-muted">
          {role}
        </span>
        <button
          type="button"
          className="cursor-pointer border-0 bg-none p-0 font-sans text-body-sm font-semibold whitespace-nowrap text-secondary"
          onClick={() => void logout().then(() => navigate({ to: '/login', replace: true }))}
        >
          ออกจากระบบ
        </button>
      </div>
    </header>
  )
}
