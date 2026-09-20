import {
  FloorPlanIcon,
  OverviewIcon,
  PatientsIcon,
  ServicePointsIcon,
  type NavItem,
} from '@/design-system'
import { useLocation, useNavigate } from '@tanstack/react-router'

/** The staff console's four destinations — identical on mobile and desktop. */
export const staffNavItems: NavItem[] = [
  { id: 'overview', label: 'ภาพรวม', icon: <OverviewIcon /> },
  { id: 'service-points', label: 'จุดบริการ', icon: <ServicePointsIcon /> },
  { id: 'floor-plan', label: 'ผังอาคาร', icon: <FloorPlanIcon /> },
  { id: 'patients', label: 'ผู้ป่วย', icon: <PatientsIcon /> },
]

/** The rail has room for longer labels than the tab bar does. */
export const staffRailItems: NavItem[] = [
  { id: 'overview', label: 'ภาพรวม', icon: <OverviewIcon size={18} /> },
  { id: 'service-points', label: 'ผังจุดบริการ', icon: <ServicePointsIcon size={18} /> },
  { id: 'floor-plan', label: 'ผังอาคาร', icon: <FloorPlanIcon size={18} /> },
  { id: 'patients', label: 'ผู้ป่วยวันนี้', icon: <PatientsIcon size={18} /> },
]

/**
 * EXECUTIVE sees only ภาพรวม — the dashboard is their whole console (#87);
 * the queue and mapping tools stay with STAFF/ADMIN. The API enforces the
 * same split (403), so hiding items is affordance, not security.
 */
export function navItemsForRole(items: NavItem[], roles: string[] | undefined): NavItem[] {
  const operational = roles?.includes('STAFF') || roles?.includes('ADMIN')
  if (roles?.includes('EXECUTIVE') && !operational) {
    return items.filter((item) => item.id === 'overview')
  }
  return items
}

/** Thai role label from the signed-in identity's token — the token decides, not the screen. */
export function staffRoleLabel(roles: string[] | undefined): string {
  if (roles?.includes('ADMIN')) return 'ผู้ดูแลระบบ'
  if (roles?.includes('EXECUTIVE')) return 'ผู้บริหาร'
  return 'เจ้าหน้าที่'
}

const paths = {
  overview: '/staff/overview',
  'service-points': '/staff/service-points',
  'floor-plan': '/staff/floor-plan',
  patients: '/staff/patients',
} as const

type StaffNavId = keyof typeof paths

/**
 * Which tab is active is a fact about the URL, not component state. The design
 * system's BottomTabBar and SideRail stay router-agnostic — they only ever see
 * an `activeId` and an `onSelect`.
 */
export function useStaffNav() {
  const navigate = useNavigate()
  const pathname = useLocation({ select: (location) => location.pathname })
  const ids = Object.keys(paths) as StaffNavId[]

  return {
    activeId: ids.find((id) => pathname.startsWith(paths[id])) ?? 'overview',
    onSelect: (id: string) => navigate({ to: paths[id as StaffNavId] }),
  }
}
