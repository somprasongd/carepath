import { createFileRoute } from '@tanstack/react-router'
import { ServicePoints } from './-ServicePoints'

export const Route = createFileRoute('/staff/service-points')({
  component: () => (
    <div className="h-dvh">
      <ServicePoints />
    </div>
  ),
})
