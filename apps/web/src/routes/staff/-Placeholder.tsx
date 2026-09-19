import { InfoNote, Meta, PageTitle, staffTitle } from '@/design-system'
import { StaffShell } from './-StaffShell'

/**
 * A destination the staff nav can reach but the MVP has not built yet. An
 * honest empty state beats a tab that goes nowhere — see docs/requirements/
 * mvp-scope.md for what is deliberately out of scope.
 */
export function StaffPlaceholder({ title, note }: { title: string; note: string }) {
  return (
    <StaffShell>
      <PageTitle className={staffTitle}>{title}</PageTitle>
      <Meta className="mt-1.5 mb-4 @7xl:mb-5">ยังไม่อยู่ในขอบเขต MVP</Meta>
      <div className="@7xl:max-w-[900px]">
        <InfoNote>{note}</InfoNote>
      </div>
    </StaffShell>
  )
}
