import {
  BottomTabBar,
  InfoNote,
  Meta,
  PageTitle,
  Screen,
  ScreenDock,
  ScreenScroll,
  ServicePointMappingCard,
} from '@/design-system'
import { servicePointMappings } from '@/mocks/demo-data'
import { staffNavItems, useStaffNav } from './-nav'

/**
 * เจ้าหน้าที่ · ผังจุดบริการ (มือถือ) — the care step → place links, one card
 * each. A place with no navigation node stays listed and says so.
 */
export function ServicePointsMobile() {
  const nav = useStaffNav()

  return (
    <Screen variant="staff-mobile">
      <header style={{ padding: '22px var(--cp-gutter) 6px', flexShrink: 0 }}>
        <PageTitle compact>ผังจุดบริการ</PageTitle>
        <div style={{ marginTop: 'var(--cp-space-xs)' }}>
          <Meta>เชื่อมโยงบริการทางคลินิกกับตำแหน่งจริงในอาคาร</Meta>
        </div>
      </header>

      <ScreenScroll style={{ padding: '14px var(--cp-gutter) 96px' }}>
        <div style={{ marginBottom: 18 }}>
          <InfoNote>ห้องที่ยังไม่มีโหนดนำทางจะเลือกได้ แต่จะยังไม่แสดงเส้นทางให้ผู้ป่วย</InfoNote>
        </div>

        <div className="cp-stack">
          {servicePointMappings.map((mapping) => (
            <ServicePointMappingCard key={mapping.service} {...mapping} />
          ))}
        </div>
      </ScreenScroll>

      <ScreenDock>
        <BottomTabBar items={staffNavItems} {...nav} />
      </ScreenDock>
    </Screen>
  )
}
