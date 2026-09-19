import { createFileRoute, redirect } from '@tanstack/react-router'

/**
 * The product's entry point is the patient journey (the LINE LIFF view);
 * the staff console lives under /staff and the component catalogue under
 * /design.
 */
export const Route = createFileRoute('/')({
  beforeLoad: () => {
    throw redirect({ to: '/patient/journey' })
  },
})
