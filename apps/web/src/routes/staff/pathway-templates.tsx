import { createFileRoute } from '@tanstack/react-router'
import { PathwayTemplates } from './-PathwayTemplates'

export const Route = createFileRoute('/staff/pathway-templates')({
  component: () => (
    <div className="h-dvh">
      <PathwayTemplates />
    </div>
  ),
})
