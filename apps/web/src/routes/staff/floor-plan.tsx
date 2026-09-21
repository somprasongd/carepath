import { createFileRoute } from '@tanstack/react-router'
import { FloorPlan } from './-FloorPlan'

export const Route = createFileRoute('/staff/floor-plan')({
  component: FloorPlan,
})
