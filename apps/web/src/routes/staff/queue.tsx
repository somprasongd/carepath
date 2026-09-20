import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Queue } from './-Queue'

/**
 * `?sp=<service point id>` — the station this console is working. Optional:
 * it defaults to the first assigned point and the picker keeps the URL in
 * sync, so a workstation's screen is bookmarkable as-is.
 */
export const Route = createFileRoute('/staff/queue')({
  validateSearch: (search: Record<string, unknown>): { sp?: string } => ({
    sp: typeof search.sp === 'string' && search.sp !== '' ? search.sp : undefined,
  }),
  component: QueueRoute,
})

function QueueRoute() {
  const navigate = useNavigate()
  const { sp } = Route.useSearch()

  return (
    <div className="h-dvh">
      <Queue
        servicePointId={sp}
        onServicePointChange={(servicePointId) =>
          void navigate({ to: '/staff/queue', search: { sp: servicePointId }, replace: true })
        }
      />
    </div>
  )
}
