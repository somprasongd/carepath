import type { ReactNode } from 'react'
import { useNavigate } from '@tanstack/react-router'

/**
 * Header for a single-task staff screen (registration, the queue console) —
 * a dedicated workstation view, not part of the four-tab staff console, so it
 * carries no side rail or tab bar. Only "ออกจากระบบ" (sign out, back to role
 * select) and whatever identifies the task belong here.
 */
export function TaskHeader({ role, station }: { role: string; station?: ReactNode }) {
  const navigate = useNavigate()

  return (
    <header className="flex shrink-0 items-center justify-between gap-4 border-b border-line bg-surface px-gutter py-4 @7xl:px-gutter-desktop">
      <div className="flex items-center gap-3.5">
        <span className="font-code text-[16px] font-bold text-ink">CarePath</span>
        {station}
      </div>
      <div className="flex items-center gap-4">
        <span className="rounded-full bg-neutral px-3.5 py-1.5 font-sans text-caption font-normal text-ink-muted">
          {role}
        </span>
        <button
          type="button"
          className="cursor-pointer border-0 bg-none p-0 font-sans text-body-sm font-semibold text-secondary"
          onClick={() => navigate({ to: '/login' })}
        >
          ออกจากระบบ
        </button>
      </div>
    </header>
  )
}
