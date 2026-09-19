import {
  AppBar,
  AttentionCard,
  BottomTabBar,
  Meta,
  PageTitle,
  RefPill,
  Screen,
  ScreenDock,
  ScreenScroll,
  SectionTitle,
  ServicePointRow,
  StatCard,
} from '@/design-system'
import { attentionList, lastUpdated, servicePointLoad } from '@/mocks/demo-data'
import { staffNavItems, staffRole, useStaffNav } from './-nav'

/** เจ้าหน้าที่ · ภาพรวม (มือถือ) — everything collapsed to one column of cards. */
export function OverviewMobile() {
  const nav = useStaffNav()
  const busiest = servicePointLoad.find((sp) => sp.load === 'busy')

  return (
    <Screen variant="staff-mobile">
      <AppBar
        wordmark={
          <>
            CarePath <span className="cp-appbar__wordmark-sub">· เจ้าหน้าที่</span>
          </>
        }
        trailing={<RefPill>{staffRole}</RefPill>}
      />

      <ScreenScroll>
        <div style={{ margin: '10px 0 2px' }}>
          <PageTitle compact>ภาพรวมการไหลของผู้ป่วยวันนี้</PageTitle>
        </div>
        <div style={{ marginBottom: 'var(--cp-space-xl)' }}>
          <Meta>อัปเดตล่าสุด {lastUpdated}</Meta>
        </div>

        <div className="cp-grid-2" style={{ marginBottom: 28 }}>
          <StatCard compact label="กำลังใช้บริการ" value="24" unit="ราย" />
          <StatCard
            compact
            textValue
            label="จุดที่รอมากสุด"
            value={busiest?.name ?? '—'}
            note={busiest ? `${busiest.waiting} คนในคิว` : undefined}
          />
          <StatCard compact label="เวลารอเฉลี่ย" value="11" unit="นาที" />
          <StatCard compact tone="attention" label="ไม่ทราบตำแหน่ง" value="3" unit="ราย" />
        </div>

        <SectionTitle compact>จุดบริการวันนี้</SectionTitle>
        <div style={{ marginBottom: 28 }}>
          {servicePointLoad.slice(0, 4).map((sp) => (
            <ServicePointRow
              key={sp.code}
              zone={sp.zone}
              code={sp.code}
              name={sp.name}
              waiting={sp.waiting}
              busy={sp.load === 'busy'}
            />
          ))}
        </div>

        <SectionTitle compact>ผู้ป่วยที่ต้องช่วยเหลือ</SectionTitle>
        <div className="cp-stack">
          {attentionList.map((item) => (
            <AttentionCard key={item.visitRef} {...item} />
          ))}
        </div>
      </ScreenScroll>

      <ScreenDock>
        <BottomTabBar items={staffNavItems} {...nav} />
      </ScreenDock>
    </Screen>
  )
}
