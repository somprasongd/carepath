import { createFileRoute } from '@tanstack/react-router'
import { Overview } from './-Overview'

export const Route = createFileRoute('/staff/overview')({
  component: () => (
    <div className="h-dvh">
      <Overview />
    </div>
  ),
})
