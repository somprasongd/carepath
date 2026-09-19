import { Outlet, createRootRoute } from '@tanstack/react-router'

/**
 * `cp-root` carries the base font and box model for everything the app renders.
 * Screens supply their own frame; this route deliberately adds no chrome, so a
 * patient LIFF view and the staff console can look nothing like each other.
 */
export const Route = createRootRoute({
  component: () => (
    <div className="cp-root">
      <Outlet />
    </div>
  ),
})
