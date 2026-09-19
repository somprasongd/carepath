import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { DEFAULT_VISIT_ID, thaiStepTitle, useVisit, type VisitView } from '@/features/visit'
import { NavigateScreen, type NavigateDestination } from './-NavigateScreen'

/** `?visit=<id>` — same parameter as the journey screen, forwarded by its CTA. */
export const Route = createFileRoute('/patient/navigate')({
  validateSearch: (search: Record<string, unknown>): { visit?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
  }),
  component: PatientNavigateRoute,
})

function PatientNavigateRoute() {
  const navigate = useNavigate()
  const { visit } = Route.useSearch()
  const visitId = visit ?? DEFAULT_VISIT_ID

  // Same query key as the journey screen, so this is a cache read, not a
  // second round trip.
  const { data, isPending } = useVisit(visitId)

  return (
    <div className="h-dvh">
      <NavigateScreen
        destination={destinationFor(data, isPending)}
        onBack={() => navigate({ to: '/patient/journey', search: { visit: visitId } })}
      />
    </div>
  )
}

function destinationFor(data: VisitView | undefined, isPending: boolean): NavigateDestination {
  if (isPending) return { title: 'กำลังโหลดจุดหมาย…', subtitle: '' }
  if (!data?.next) return { title: 'จุดบริการของคุณ', subtitle: '', placeId: 'unknown' }

  const next = data.next
  const step = data.steps.find((s) => s.sequence === next.sequence)
  const title = thaiStepTitle(step?.serviceCode ?? '')
  const servicePoint = next.servicePoint

  return {
    title: `เส้นทางไป${title}`,
    subtitle: servicePoint
      ? `${servicePoint.name} · ${servicePoint.placeId}`
      : (step?.serviceCode ?? ''),
    placeId: servicePoint?.placeId,
  }
}
