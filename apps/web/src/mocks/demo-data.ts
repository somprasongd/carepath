/**
 * Static stand-in data for the reference screens shown on /design and the
 * staff console, whose API endpoints do not exist yet.
 *
 * The patient journey/navigate screens no longer read this — they render the
 * live visit from features/visit (only their schematic floor plan is still
 * here, until /api/v1/navigation/route is implemented).
 */

import type { JourneyStep, MapRoom, RouteStatus, Zone } from '../design-system'

export const visitRef = 'V-2384'

export const journeySteps: JourneyStep[] = [
  { id: 'reg', state: 'done', title: 'ลงทะเบียน', meta: 'เสร็จสิ้น · REG-01' },
  { id: 'opd', state: 'done', title: 'ตรวจที่ห้องตรวจ OPD', meta: 'เสร็จสิ้น · OPD-EXAM-01' },
  {
    id: 'lab',
    state: 'current',
    title: 'เจาะเลือด',
    meta: 'ห้องเจาะเลือด · ชั้น 2 · LAB-01',
    queue: { number: '12', wait: 'รอประมาณ 8 นาที' },
  },
  {
    id: 'pharmacy',
    state: 'next',
    title: 'รับยา',
    meta: 'ห้องยา · ชั้น 1 · เดิน 3 นาที · PHARMACY-01',
  },
  { id: 'cashier', state: 'pending', title: 'การเงิน', meta: 'รอดำเนินการ · CASHIER-01' },
]

export const floor1Rooms: MapRoom[] = [
  { id: 'reg', zone: 'public', label: 'รับลงทะเบียน', x: 0, y: 0, width: 76, height: 60 },
  { id: 'cashier', zone: 'public', label: 'การเงิน', x: 80, y: 0, width: 52, height: 60 },
  { id: 'wait', zone: 'public', label: 'จุดรอ', x: 136, y: 0, width: 84, height: 60 },
  {
    id: 'pharmacy',
    zone: 'pharmacy',
    label: 'ห้องยา',
    code: 'PHARMACY-01',
    x: 224,
    y: 0,
    width: 60,
    height: 60,
    destination: true,
  },
  { id: 'support', zone: 'support', label: 'Support', x: 288, y: 0, width: 52, height: 60 },
]

export const floor1Corridor = { x: 0, y: 68, width: 340, height: 48, label: 'ทางเดินหลัก' }
export const pharmacyRoute = 'M170 92 L254 92 L254 46'
export const pharmacyRouteEnd = { x: 254, y: 46 }
export const currentLocation = { x: 170, y: 92, label: 'คุณอยู่ที่นี่' }

export const walkingSteps = [
  'เดินตรงไปตามทางเดินหลัก ประมาณ 45 เมตร',
  'เลี้ยวขวาเข้าห้องยา (Pharmacy) ทางขวามือ',
]

export type ServicePointLoad = {
  code: string
  name: string
  zone: Zone
  waiting: number
  averageWait: string
  load: 'normal' | 'busy'
}

export const servicePointLoad: ServicePointLoad[] = [
  { code: 'REG-01', name: 'Reception', zone: 'public', waiting: 3, averageWait: '4 นาที', load: 'normal' },
  { code: 'LAB-01', name: 'เจาะเลือด', zone: 'diagnostic', waiting: 8, averageWait: '9 นาที', load: 'busy' },
  {
    code: 'PHARMACY-01',
    name: 'ห้องยา',
    zone: 'pharmacy',
    waiting: 6,
    averageWait: '6 นาที',
    load: 'normal',
  },
  { code: 'XRAY-01', name: 'เอกซเรย์', zone: 'diagnostic', waiting: 2, averageWait: '5 นาที', load: 'normal' },
  { code: 'CASHIER-01', name: 'การเงิน', zone: 'public', waiting: 4, averageWait: '3 นาที', load: 'normal' },
]

export type ServicePointMapping = {
  service: string
  serviceName: string
  target: string
  note?: string
  zone: Zone
  status: RouteStatus
}

export const servicePointMappings: ServicePointMapping[] = [
  {
    service: 'REG',
    serviceName: 'Registration',
    target: 'REG-01 · Reception · ชั้น 1',
    zone: 'public',
    status: 'routable',
  },
  {
    service: 'CASHIER',
    serviceName: 'Cashier',
    target: 'CASHIER-01 · การเงิน · ชั้น 1',
    zone: 'public',
    status: 'routable',
  },
  {
    service: 'PHARMACY',
    serviceName: 'Pharmacy',
    target: 'PHARMACY-01 · ห้องยา · ชั้น 1',
    zone: 'pharmacy',
    status: 'routable',
  },
  {
    service: 'LAB',
    serviceName: 'Blood collection',
    target: 'LAB-01 · ห้องเจาะเลือด · ชั้น 2',
    note: 'ใช้ห้องเจาะเลือดร่วมกัน — อาคารนี้ไม่มีห้องแล็บแยก',
    zone: 'diagnostic',
    status: 'routable',
  },
  {
    service: 'XRAY',
    serviceName: 'Radiography',
    target: 'XRAY-01 · ห้องเอกซเรย์ · ชั้น 1',
    zone: 'diagnostic',
    status: 'routable',
  },
  {
    service: 'REHAB',
    serviceName: 'Rehabilitation',
    target: 'REHAB-01 · ห้องกายภาพบำบัด · ชั้น 1',
    zone: 'rehab',
    status: 'not-routable',
  },
  {
    service: 'IPD',
    serviceName: 'Inpatient ward',
    target: 'IPD-01 · หอผู้ป่วยใน · ชั้น 1',
    zone: 'ipd',
    status: 'not-routable',
  },
]

export type AttentionItem = {
  visitRef: string
  step: string
  reason: string
  urgent?: boolean
  actionLabel: string
}

export const attentionList: AttentionItem[] = [
  {
    visitRef: 'V-2210',
    step: 'เจาะเลือด',
    reason: 'ไม่ทราบตำแหน่งมา 6 นาที',
    urgent: true,
    actionLabel: 'ช่วยนำทาง',
  },
  {
    visitRef: 'V-2197',
    step: 'คัดกรอง (Triage)',
    reason: 'รอเกิน 22 นาทีที่ TRIAGE-01',
    actionLabel: 'ตรวจสอบคิว',
  },
]

export const lastUpdated = '09:42 น.'
export const consoleDate = 'อังคาร 19 ก.ย. 2569'
