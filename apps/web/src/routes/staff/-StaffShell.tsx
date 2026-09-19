import { AppBar, BottomTabBar, Screen, ScreenBody, ScreenDock, SideRail, WordmarkSub } from '@/design-system'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import { useNavigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { staffNavItems, staffRailItems, useStaffNav } from './-nav'

/**
 * The staff console's one layout: a fixed rail from ~1280px (DESIGN.md's
 * desktop breakpoint) up, an app bar + bottom tab bar below it. Every
 * `/staff/*` route renders its content once, through this shell, instead of
 * shipping a separate mobile and desktop screen. The signed-in identity and
 * its sign-out live here so every screen gets them for free (ADR-0010 §9).
 */
export function StaffShell({
  trailing,
  children,
}: {
  trailing?: ReactNode
  children: ReactNode
}) {
  const nav = useStaffNav()
  const { identity, logout } = useStaffAuth()
  const navigate = useNavigate()

  const signOut = () => {
    void logout().then(() => navigate({ to: '/login', replace: true }))
  }

  return (
    <Screen variant="staff">
      <SideRail
        items={staffRailItems}
        {...nav}
        role={identity?.displayName ?? 'เจ้าหน้าที่'}
        onSignOut={signOut}
        className="hidden @7xl:flex"
      />

      <div className="relative flex flex-1 flex-col overflow-hidden">
        <AppBar
          className="@7xl:hidden"
          wordmark={
            <>
              CarePath <WordmarkSub>· เจ้าหน้าที่</WordmarkSub>
            </>
          }
          trailing={
            <>
              {trailing}
              <button
                type="button"
                className="shrink-0 cursor-pointer border-0 bg-none p-0 font-sans text-caption font-semibold text-ink-muted underline underline-offset-2"
                onClick={signOut}
              >
                ออกจากระบบ
              </button>
            </>
          }
        />

        <ScreenBody>{children}</ScreenBody>

        <ScreenDock className="@7xl:hidden">
          <BottomTabBar items={staffNavItems} {...nav} />
        </ScreenDock>
      </div>
    </Screen>
  )
}
