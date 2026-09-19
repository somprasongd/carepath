import {
  AppBar,
  BottomTabBar,
  InfoNote,
  Meta,
  PageTitle,
  Screen,
  ScreenDock,
  ScreenMain,
  ScreenScroll,
  SideRail,
} from '@/design-system'
import { staffNavItems, staffRailItems, staffRole, useStaffNav } from './-nav'

/**
 * A destination the staff nav can reach but the MVP has not built yet. An
 * honest empty state beats a tab that goes nowhere — see docs/requirements/
 * mvp-scope.md for what is deliberately out of scope.
 */
export function StaffPlaceholder({ title, note }: { title: string; note: string }) {
  const nav = useStaffNav()

  return (
    <>
      <div className="cp-viewport cp-at-mobile">
        <Screen variant="staff-mobile">
          <AppBar
            wordmark={
              <>
                CarePath <span className="cp-appbar__wordmark-sub">· เจ้าหน้าที่</span>
              </>
            }
          />
          <ScreenScroll>
            <div style={{ margin: '10px 0 2px' }}>
              <PageTitle compact>{title}</PageTitle>
            </div>
            <div style={{ marginBottom: 'var(--cp-space-xl)' }}>
              <Meta>ยังไม่อยู่ในขอบเขต MVP</Meta>
            </div>
            <InfoNote>{note}</InfoNote>
          </ScreenScroll>
          <ScreenDock>
            <BottomTabBar items={staffNavItems} {...nav} />
          </ScreenDock>
        </Screen>
      </div>

      <div className="cp-viewport cp-at-desktop">
        <Screen variant="desktop">
          <SideRail items={staffRailItems} {...nav} role={staffRole} />
          <ScreenMain>
            <PageTitle>{title}</PageTitle>
            <div style={{ marginTop: 6, marginBottom: 'var(--cp-space-xl)' }}>
              <Meta>ยังไม่อยู่ในขอบเขต MVP</Meta>
            </div>
            <div style={{ maxWidth: 900 }}>
              <InfoNote>{note}</InfoNote>
            </div>
          </ScreenMain>
        </Screen>
      </div>
    </>
  )
}
