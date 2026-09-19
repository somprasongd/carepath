import { App } from '@/App'
import { createFileRoute } from '@tanstack/react-router'

/**
 * The one screen that talks to the real API today — a smoke test for the
 * CarePath ↔ Mock HIS round trip. features/visit replaces it at step 4.
 */
export const Route = createFileRoute('/')({
  component: App,
})
