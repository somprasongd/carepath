import { createFileRoute } from '@tanstack/react-router'
import { OverviewDesktop } from './-OverviewDesktop'
import { OverviewMobile } from './-OverviewMobile'

export const Route = createFileRoute('/staff/overview')({
  component: StaffOverviewRoute,
})

/**
 * Transitional: one URL, two hand-built layouts, picked by breakpoint. Step 3
 * of the Tailwind migration merges them into one responsive screen.
 */
function StaffOverviewRoute() {
  return (
    <>
      <div className="cp-viewport cp-at-mobile">
        <OverviewMobile />
      </div>
      <div className="cp-viewport cp-at-desktop">
        <OverviewDesktop />
      </div>
    </>
  )
}
