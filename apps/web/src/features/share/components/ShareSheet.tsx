import { useEffect, useRef, useState } from 'react'
import { Button, Divider, Meta } from '@/design-system'
import { useCreateShareLink, useStopSharing } from '../queries'
import { shareExpiryLabel } from '../shared-view'

/**
 * แชร์ความคืบหน้าให้ญาติ — the share bottom sheet (#90). Opened from the
 * journey screen's ghost button, never the sticky bar: the one orange action
 * on a patient screen stays "นำทาง" (apps/web AGENTS.md design rules).
 *
 * The sheet mints a link on open, shows it with a copy affordance and its
 * expiry in plain Thai, and can stop sharing — revoking every active link at
 * once, then minting a fresh one so the cap (5 active) never blocks the next
 * share. The token is composed into a URL here and nowhere else; it travels
 * in the fragment after /shared so no server or proxy ever sees it
 * (ADR-0011 §5).
 */
export function ShareSheet({ visitId, onClose }: { visitId: string; onClose: () => void }) {
  const create = useCreateShareLink(visitId)
  const stop = useStopSharing(visitId)
  const [copied, setCopied] = useState(false)
  const [stopped, setStopped] = useState(false)
  const minted = useRef(false)

  useEffect(() => {
    if (minted.current) return
    minted.current = true
    create.mutate()
  }, [create])

  const stopSharing = () => {
    stop.mutate(undefined, {
      onSuccess: () => {
        setStopped(true)
        // A stopped sheet mints a new link immediately — the patient asked to
        // stop, not to close the door on sharing again.
        create.mutate()
      },
    })
  }

  const link = create.data ? `${window.location.origin}/shared#${create.data.token}` : null

  const copy = async () => {
    if (!link) return
    if (await copyText(link)) {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col justify-end"
      role="dialog"
      aria-modal="true"
      aria-label="แชร์ความคืบหน้าให้ญาติ"
    >
      {/* The scrim is a button so tapping anywhere above the sheet closes it. */}
      <button
        type="button"
        aria-label="ปิด"
        className="flex-1 cursor-default bg-ink/45"
        onClick={onClose}
      />
      <div className="rounded-t-xl bg-surface px-gutter pt-3 pb-6 shadow-sheet">
        <div className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
        <div className="mt-3.5 font-sans text-body-md font-bold text-ink">
          แชร์ความคืบหน้าให้ญาติ
        </div>
        <Meta className="mt-1">
          ญาติเปิดลิงก์นี้เห็นเฉพาะขั้นตอนที่กำลังอยู่ ไม่เห็นข้อมูลอื่นของคุณ
        </Meta>

        <Divider />

        {create.isPending && <Meta>กำลังสร้างลิงก์…</Meta>}

        {create.isError && (
          <div className="flex flex-col gap-2.5">
            <Meta>{createErrorMessage(create.error)}</Meta>
            {create.error.status === 409 ? (
              <Button variant="secondary" onClick={stopSharing} disabled={stop.isPending}>
                {stop.isPending ? 'กำลังหยุดแชร์…' : 'หยุดแชร์ลิงก์เดิมก่อน'}
              </Button>
            ) : (
              <Button variant="ghost" onClick={() => create.mutate()}>
                ลองสร้างใหม่
              </Button>
            )}
          </div>
        )}

        {link && create.data && (
          <div className="flex flex-col gap-3">
            <p className="m-0 break-all rounded-lg border border-line bg-neutral px-3 py-2.5 font-sans text-body-sm text-ink">
              {link}
            </p>
            <div className="flex flex-wrap items-center gap-2.5">
              <Button variant="secondary" onClick={() => void copy()}>
                {copied ? 'คัดลอกแล้ว' : 'คัดลอกลิงก์'}
              </Button>
              <Button variant="ghost" onClick={stopSharing} disabled={stop.isPending}>
                {stop.isPending ? 'กำลังหยุดแชร์…' : 'หยุดแชร์'}
              </Button>
            </div>
            <Meta>{shareExpiryLabel(create.data.expiresAt)}</Meta>
            {stopped && !stop.isPending && (
              <Meta className="text-ink">หยุดแชร์ลิงก์เดิมแล้ว — ลิงก์ด้านบนคือลิงก์ใหม่</Meta>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function createErrorMessage(error: { status: number }): string {
  if (error.status === 409) return 'มีลิงก์ที่ยังใช้ได้มากเกินไป — หยุดแชร์ลิงก์เดิมก่อนแล้วลองใหม่'
  if (error.status === 401) return 'เข้าสู่ระบบใหม่ก่อนจึงจะแชร์ได้'
  return 'สร้างลิงก์ไม่สำเร็จ — ลองอีกครั้ง'
}

/** Clipboard with the legacy fallback for webviews without the async API. */
async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // Fall through to the legacy path.
  }
  try {
    const area = document.createElement('textarea')
    area.value = text
    area.style.position = 'fixed'
    area.style.opacity = '0'
    document.body.appendChild(area)
    area.select()
    const ok = document.execCommand('copy')
    area.remove()
    return ok
  } catch {
    return false
  }
}
