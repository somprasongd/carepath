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

export const staffRole = 'ประชาสัมพันธ์'

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
