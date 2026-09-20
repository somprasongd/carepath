import { useState, type FormEvent } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Button, Card, PageTitle, TextField } from '@/design-system'

/**
 * ค้นหาการนัดหมายด้วยหมายเลขการรักษา (VN) — the patient side's front door.
 * Replaces the old demo auto-entry: the visit must be named before any
 * journey renders. What the patient types is the HIS visit reference the
 * journey API is keyed on (`?visit=` on the patient routes) — VISIT-001 /
 * VISIT-002 in the demo data, the hospital's real VN once a production HIS
 * feeds it. A wrong number surfaces the journey screen's not-found state.
 */
export function VisitEntryScreen() {
  const navigate = useNavigate()
  const [vn, setVn] = useState('')
  const [error, setError] = useState<string | undefined>()

  const submit = (event: FormEvent) => {
    event.preventDefault()
    const value = vn.trim()
    if (value === '') {
      setError('กรุณากรอกหมายเลขการรักษา')
      return
    }
    navigate({ to: '/patient/journey', search: { visit: value } })
  }

  return (
    <div className="@container h-full w-full">
      <div className="flex h-full items-center justify-center overflow-y-auto bg-neutral px-gutter py-12">
        <Card radius="lg" padding="xl" className="w-full max-w-[440px]">
          <div className="mb-9 flex items-baseline gap-2">
            <span className="font-code text-[20px] font-bold text-ink">CarePath</span>
            <span className="font-sans text-body-sm text-ink-muted">ผู้ป่วย</span>
          </div>

          <PageTitle className="mb-1.5">ค้นหาการนัดหมาย</PageTitle>
          <p className="mt-0 mb-7 font-sans text-body-sm text-ink-muted">
            กรอกหมายเลขการรักษา (VN) จากสลิกของโรงพยาบาลเพื่อดูแผนการรักษาของคุณ
          </p>

          <form onSubmit={submit}>
            <div className="mb-5 grid grid-cols-1 gap-4">
              <TextField
                label="หมายเลขการรักษา (VN)"
                placeholder="เช่น VISIT-001"
                value={vn}
                onChange={(e) => setVn(e.target.value)}
              />
            </div>

            {error && (
              <p role="alert" className="mb-4 font-sans text-body-sm text-warning">
                {error}
              </p>
            )}

            <Button type="submit" variant="secondary" block>
              ดูแผนการรักษา
            </Button>
          </form>
        </Card>
      </div>
    </div>
  )
}
