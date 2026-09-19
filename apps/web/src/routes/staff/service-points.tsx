import { createFileRoute } from '@tanstack/react-router'
import { ServicePointsDesktop } from './-ServicePointsDesktop'
import { ServicePointsMobile } from './-ServicePointsMobile'

export const Route = createFileRoute('/staff/service-points')({
  component: StaffServicePointsRoute,
})

/** See the note in overview.tsx — the same transitional two-layout split. */
function StaffServicePointsRoute() {
  return (
    <>
      <div className="cp-viewport cp-at-mobile">
        <ServicePointsMobile />
      </div>
      <div className="cp-viewport cp-at-desktop">
        <ServicePointsDesktop />
      </div>
    </>
  )
}
