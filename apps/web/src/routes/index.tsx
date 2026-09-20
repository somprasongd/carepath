import { createFileRoute, redirect } from '@tanstack/react-router'

/**
 * The front door is the staff console's login (FR-18). Patients reach their
 * journey via LINE — or, outside LINE, by naming their visit number (VN) on
 * /patient/journey. Nobody lands inside the demo patient by default.
 */
export const Route = createFileRoute('/')({
  beforeLoad: () => {
    throw redirect({ to: '/login' })
  },
})
