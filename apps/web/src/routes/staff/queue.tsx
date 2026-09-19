import { createFileRoute } from '@tanstack/react-router'
import { Queue } from './-Queue'

export const Route = createFileRoute('/staff/queue')({
  component: () => (
    <div className="h-dvh">
      <Queue />
    </div>
  ),
})
