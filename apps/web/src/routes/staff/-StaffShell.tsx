import { AppBar, BottomTabBar, Screen, ScreenBody, ScreenDock, SideRail, WordmarkSub } from '@/design-system'
import type { ReactNode } from 'react'
import { staffNavItems, staffRailItems, staffRole, useStaffNav } from './-nav'

/**
 * The staff console's one layout: a fixed rail from ~1280px (DESIGN.md's
 * desktop breakpoint) up, an app bar + bottom tab bar below it. Every
 * `/staff/*` route renders its content once, through this shell, instead of
 * shipping a separate mobile and desktop screen.
 */
export function StaffShell({
  trailing,
  children,
}: {
  trailing?: ReactNode
  children: ReactNode
}) {
  const nav = useStaffNav()

  return (
    <Screen variant="staff">
      <SideRail items={staffRailItems} {...nav} role={staffRole} className="hidden @7xl:flex" />

      <div className="relative flex flex-1 flex-col overflow-hidden">
        <AppBar
          className="@7xl:hidden"
          wordmark={
            <>
              CarePath <WordmarkSub>· เจ้าหน้าที่</WordmarkSub>
            </>
          }
          trailing={trailing}
        />

        <ScreenBody>{children}</ScreenBody>

        <ScreenDock className="@7xl:hidden">
          <BottomTabBar items={staffNavItems} {...nav} />
        </ScreenDock>
      </div>
    </Screen>
  )
}
