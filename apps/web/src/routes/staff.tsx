import { Outlet, createFileRoute } from '@tanstack/react-router'
import { RequireStaffAuth } from '@/auth/RequireStaffAuth'
import { StaffAuthProvider } from '@/auth/StaffAuthProvider'

/**
 * Layout route for everything under /staff — the console. Every screen
 * renders through the staff session: restored from the persisted refresh
 * token on mount, bounced to /login when it is gone (ADR-0010 §9).
 */
export const Route = createFileRoute('/staff')({
  component: () => (
    <StaffAuthProvider>
      <RequireStaffAuth>
        <Outlet />
      </RequireStaffAuth>
    </StaffAuthProvider>
  ),
})
