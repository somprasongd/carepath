import { createFileRoute } from '@tanstack/react-router'
import { Register } from './-Register'

export const Route = createFileRoute('/staff/register')({
  component: () => (
    <div className="h-dvh">
      <Register />
    </div>
  ),
})
