import { Card, CheckIcon } from '@/design-system'

/**
 * How a finished visit reads on the patient home screen (#34): a summary
 * card above the all-done rail, no action bar. Informational only — ink on
 * surface, because orange is reserved for the route CTA (DESIGN.md).
 */
export function VisitOutcomeCard({ outcome }: { outcome: 'completed' | 'cancelled' }) {
  if (outcome === 'cancelled') {
    return (
      <Card radius="md" padding="md" className="mb-5">
        <div className="mb-1 font-sans text-caption text-ink-muted">การมาโรงพยาบาลนี้</div>
        <div className="mb-1 text-[18px] font-bold text-ink">ถูกยกเลิก</div>
        <p className="m-0 font-sans text-body-sm text-ink-muted">
          หากคุณมีข้อสงสัย โปรดติดต่อเจ้าหน้าที่ที่จุดบริการ
        </p>
      </Card>
    )
  }

  return (
    <Card radius="md" padding="md" className="mb-5">
      <div className="mb-2 flex items-center gap-2.5">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-ink text-surface">
          <CheckIcon />
        </span>
        <span className="text-[18px] font-bold text-ink">เสร็จสิ้นทุกขั้นตอนแล้ว</span>
      </div>
      <p className="m-0 font-sans text-body-sm text-ink-muted">
        ขอบคุณที่ใช้บริการวันนี้ ขอให้คุณมีสุขภาพแข็งแรง
      </p>
    </Card>
  )
}
