import { Card, Code, InfoNote, Meta, PageTitle, SectionTitle } from '@/design-system'
import { StaffStatusBadge } from '@/features/visit/components/StaffStatusBadge'
import { stepTitle } from '@/features/visit/journey'
import { usePlanningRules } from '@/features/visit/queries'
import type { PlanningRules } from '@/features/visit/queries'
import { TaskHeader } from './-TaskHeader'

/**
 * เจ้าหน้าที่ · กฎการวางแผนการเดินของผู้ป่วย — FR-13/US-16 (#100).
 *
 * Everything on this screen is the planner describing itself: the phase
 * ladder, the order-type mapping, and worked examples computed by the real
 * Plan function, served by GET /api/v1/staff/planning-rules. Editing the
 * rules means editing planner.go (ADR-0009 — the rules are code, not data);
 * the endpoint derives from the planner directly, so this page changes with
 * it and can never show a stale table.
 *
 * The phase keys and example ids are stable codes (ADR-0012); the Thai
 * labels below are this screen's own. Staff screens pin Thai and may use
 * operational vocabulary, so these stay raw literals, not catalogs.
 */

const PHASE_LABELS: Record<string, string> = {
  REGISTRATION: 'ลงทะเบียน',
  PRE_VISIT: 'ตรวจก่อนพบแพทย์',
  CLINIC: 'พบแพทย์',
  MID_VISIT: 'ตรวจที่หมอสั่งระหว่างพบแพทย์',
  RETURN_CLINIC: 'กลับไปพบแพทย์อีกครั้ง',
  CASHIER: 'ชำระเงิน',
  PHARMACY: 'รับยา',
}

const ORDER_TYPE_LABELS: Record<string, string> = {
  LAB: 'แล็บ',
  XRAY: 'เอกซ์เรย์',
  EKG: 'EKG',
  US: 'อัลตราซาวด์',
  DRUG: 'ยา',
}

const EXAMPLE_LABELS: Record<string, string> = {
  'fresh-visit': 'ผู้ป่วย walk-in: มีแล็บก่อนพบแพทย์ และมีรายการยา',
  'clinic-round-return': 'หมอสั่งแล็บระหว่างพบแพทย์ แล้วผู้ป่วยต้องกลับมาพบอีกครั้ง',
}

const phaseLabel = (key: string) => PHASE_LABELS[key] ?? key
const orderTypeLabel = (type: string) => ORDER_TYPE_LABELS[type] ?? type
const exampleLabel = (id: string) => EXAMPLE_LABELS[id] ?? id

export function PathwayTemplates() {
  const rules = usePlanningRules()

  return (
    <div className="@container flex h-dvh flex-col bg-neutral">
      <TaskHeader role="เจ้าหน้าที่" />

      <main className="flex-1 overflow-y-auto px-gutter py-8 @7xl:px-gutter-desktop">
        <div className="mx-auto max-w-[1240px]">
          <PageTitle>กฎการวางแผนการเดินของผู้ป่วย</PageTitle>
          <Meta className="mt-1.5 mb-5">
            กฎที่ระบบใช้วางแผนจริงในขณะนี้ — ดึงจาก planner ของ API โดยตรง ไม่ใช่ข้อมูลตัวอย่าง
          </Meta>

          <div className="mb-6 max-w-[720px]">
            <InfoNote>
              หน้านี้ดูอย่างเดียว — แก้กฎคือแก้โค้ด planner (ADR-0009) แล้วหน้านี้เปลี่ยนตามอัตโนมัติ
            </InfoNote>
          </div>

          {rules.isPending ? (
            <Meta>กำลังโหลดกฎจาก API…</Meta>
          ) : rules.isError ? (
            <div className="max-w-[720px]">
              <InfoNote>
                โหลดกฎไม่สำเร็จ (สถานะ {rules.error.status}) — ตรวจว่าล็อกอิน staff และ API
                ทำงานอยู่ แล้วรีเฟรชอีกครั้ง
              </InfoNote>
            </div>
          ) : (
            <Rules rules={rules.data} />
          )}
        </div>
      </main>
    </div>
  )
}

function Rules({ rules }: { rules: PlanningRules }) {
  return (
    <>
      <div className="grid grid-cols-1 gap-6 @5xl:grid-cols-2">
        <PhaseLadder phases={rules.phases} />
        <OrderTypeMap orderTypes={rules.orderTypes} />
      </div>

      <div className="mt-6">
        <SectionTitle className="mb-1">ตัวอย่างแผนจาก planner จริง</SectionTitle>
        <Meta className="mb-4">
          สถานะที่เห็นคือผลคำนวณจาก planner ตัวจริงกับข้อมูลตัวอย่าง — ไม่ใช่ข้อความที่เขียนไว้ล่วงหน้า
        </Meta>
        <div className="grid grid-cols-1 gap-6 @5xl:grid-cols-2">
          {rules.examples.map((example) => (
            <ExampleCard key={example.id} example={example} />
          ))}
        </div>
      </div>

      <div className="mt-6 max-w-[720px]">
        <InfoNote>
          กฎบางส่วนเป็น logic ที่แสดงเป็นตารางไม่ได้ เช่น การคงประวัติขั้นตอนที่เริ่มไปแล้ว (§7)
          เงื่อนไขการปิดรอบพบแพทย์ (§4) และการห้ามถอนขั้นที่จบแล้ว — รายละเอียดเต็มอยู่ที่{' '}
          <Code>planner.go</Code> และ ADR-0009
        </InfoNote>
      </div>
    </>
  )
}

function PhaseLadder({ phases }: { phases: PlanningRules['phases'] }) {
  return (
    <Card radius="lg" padding="xl">
      <SectionTitle>ลำดับขั้นของแผน (Phase)</SectionTitle>
      <Meta className="-mt-2 mb-4">
        ขั้นตอนจะทำได้เมื่อทุกขั้นใน phase ที่ต่ำกว่าเสร็จหรือถูกยกเลิก — ขั้นใน phase
        เดียวกันไม่มีลำดับระหว่างกัน (§6)
      </Meta>

      <ol className="m-0 flex list-none flex-col p-0">
        {phases.map((phase, i) => (
          <li key={phase.key} className="flex gap-3">
            <div className="flex w-[38px] shrink-0 flex-col items-center">
              <span className="flex h-[22px] min-w-[38px] shrink-0 items-center justify-center rounded-full border border-line bg-neutral px-1 font-code text-[11px] font-bold text-ink-muted">
                {phase.number}
              </span>
              {i < phases.length - 1 && <span className="mt-0.5 min-h-5 w-0.5 flex-1 bg-line" />}
            </div>
            <div className="pb-5">
              <div className="font-sans text-body-md font-semibold text-ink">
                {phaseLabel(phase.key)}
              </div>
              <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">
                {phase.key}
              </div>
            </div>
          </li>
        ))}
      </ol>
    </Card>
  )
}

function OrderTypeMap({ orderTypes }: { orderTypes: PlanningRules['orderTypes'] }) {
  return (
    <Card radius="lg" padding="xl">
      <SectionTitle>รายการที่ HIS สั่ง → ขั้นตอนของผู้ป่วย</SectionTitle>
      <Meta className="-mt-2 mb-4">
        ทุกชนิด order ที่ planner รู้จัก และขั้นตอนที่ผู้ป่วยต้องเดินไปทำจริง
      </Meta>

      <ul className="m-0 mb-5 flex list-none flex-col gap-3 p-0">
        {orderTypes.map((row) => (
          <li
            key={row.orderType}
            className="rounded-md border border-line bg-surface px-3 py-2.5"
          >
            <div className="flex items-baseline justify-between gap-3">
              <span className="font-sans text-body-md font-semibold text-ink">
                {orderTypeLabel(row.orderType)}
              </span>
              <Code>{row.orderType}</Code>
            </div>
            <div className="mt-1 font-sans text-caption font-normal text-ink-muted">
              {row.stepKind
                ? `กลายเป็นขั้นตอน ${stepTitle({ kind: row.stepKind, clinicCode: null, round: null }, 'th')} (${row.stepKind})`
                : 'ไม่กลายเป็นขั้นตอนเดินของตัวเอง — การมีรายการยาที่ยังไม่ถูกยกเลิกทำให้ visit มีขั้นตอน "รับยา"'}
            </div>
          </li>
        ))}
      </ul>
    </Card>
  )
}

function ExampleCard({ example }: { example: PlanningRules['examples'][number] }) {
  return (
    <Card radius="lg" padding="xl">
      <SectionTitle>{exampleLabel(example.id)}</SectionTitle>
      <Meta className="-mt-2 mb-1">ข้อมูลที่ป้อน (จาก HIS)</Meta>
      <Meta className="mb-4">
        {`คลินิก: ${example.scenario.clinics.join(', ') || '—'} · รายการ: ${example.scenario.orders
          .map((order) =>
            order.orderedByClinic
              ? `${orderTypeLabel(order.orderType)} (สั่งโดย ${order.orderedByClinic})`
              : `${orderTypeLabel(order.orderType)} (สั่งตอนลงทะเบียน)`,
          )
          .join(', ')}`}
      </Meta>

      <ol className="m-0 flex list-none flex-col p-0">
        {example.steps.map((step, i) => (
          <li key={step.stepKey} className="flex gap-3">
            <div className="flex w-[22px] shrink-0 flex-col items-center">
              <span className="flex size-[22px] shrink-0 items-center justify-center rounded-full border border-line bg-neutral font-code text-[11px] font-bold text-ink-muted">
                {i + 1}
              </span>
              {i < example.steps.length - 1 && (
                <span className="mt-0.5 min-h-5 w-0.5 flex-1 bg-line" />
              )}
            </div>
            <div className="flex flex-1 items-start justify-between gap-3 pb-5">
              <div>
                <div className="font-sans text-body-md font-semibold text-ink">
                  {stepTitle(step, 'th')}
                </div>
                <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">
                  {step.stepKey}
                </div>
              </div>
              <StaffStatusBadge status={step.status} kind="step" />
            </div>
          </li>
        ))}
      </ol>
    </Card>
  )
}
