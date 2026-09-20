import { useRef, useState } from 'react'
import { ApiError } from '@/api/client'
import { format, messagesFor, useLocale, useT } from '@/i18n'
import { floorLabelFor } from '@/features/floorplan'
import { useReportLocation, useQrScanner } from '@/features/navigation'
import { useServicePoints } from '@/features/servicepoint'

/**
 * Full-screen location report for the navigate screen: the camera QR scanner
 * when the browser supports it (ADR-0004's QR source), the service-point list
 * otherwise and on demand (the MANUAL source — a picked entry node). One
 * successful report closes the overlay; the walking line and the
 * you-are-here mark redraw on the map beneath by themselves.
 */
export function ScanOverlay({
  visitId,
  initialMode,
  defaultFloorId,
  onClose,
  onReported,
}: {
  visitId: string
  /** Which affordance opened this — the button under the map or the pick link. */
  initialMode: 'camera' | 'pick'
  /** The floor the map is showing — the pick list opens on it. */
  defaultFloorId?: string
  onClose: () => void
  /** Fired once per accepted fix — the screen beneath uses it to settle UI. */
  onReported: () => void
}) {
  const t = useT()
  const [mode, setMode] = useState<'camera' | 'pick'>(initialMode)

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-ink/95 px-gutter pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1.25rem,env(safe-area-inset-bottom))]">
      <div className="flex items-center justify-between">
        <span className="font-sans text-body-md font-bold text-surface">
          {mode === 'camera' ? t('scan.title.camera') : t('scan.title.pick')}
        </span>
        <button
          type="button"
          onClick={onClose}
          className="cursor-pointer rounded-full border border-surface/40 px-3 py-1.5 font-sans text-caption font-bold text-surface"
        >
          {t('scan.close')}
        </button>
      </div>

      {mode === 'camera' ? (
        <CameraScan
          visitId={visitId}
          onReported={onReported}
          onPickInstead={() => setMode('pick')}
        />
      ) : (
        <PlaceList
          visitId={visitId}
          defaultFloorId={defaultFloorId}
          onReported={onReported}
          onScanInstead={() => setMode('camera')}
        />
      )}
    </div>
  )
}

function CameraScan({
  visitId,
  onReported,
  onPickInstead,
}: {
  visitId: string
  onReported: () => void
  onPickInstead: () => void
}) {
  const t = useT()
  const report = useReportLocation(visitId)
  const reporting = useRef(false)
  // One detection = one report; a sticker held in view fires the detector
  // continuously and must not queue a mutation per frame.
  const { videoRef, state } = useQrScanner((raw) => {
    if (reporting.current) return
    reporting.current = true
    report.mutate(
      { source: 'QR', raw },
      {
        onSuccess: onReported,
        onSettled: () => {
          reporting.current = false
        },
      },
    )
  })

  const notice =
    report.error
      ? reportErrorLabel(report.error, t)
      : state === 'unsupported'
        ? t('scan.cameraUnsupported')
        : typeof state === 'object' && 'denied' in state
          ? t('scan.cameraDenied')
          : null

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4">
      {state === 'scanning' && (
        <div className="relative aspect-square w-full max-w-80 overflow-hidden rounded-lg border border-surface/40 bg-black">
          {/* Muted + inline: the only autoplay contract iOS grants a scanner. */}
          <video ref={videoRef} muted playsInline className="h-full w-full object-cover" />
          <div className="pointer-events-none absolute inset-6 rounded-lg border-2 border-dashed border-surface/70" />
        </div>
      )}
      <p className="m-0 max-w-80 text-center font-sans text-body-sm text-surface">
        {notice ?? t('scan.cameraHint')}
      </p>
      <button
        type="button"
        onClick={onPickInstead}
        className="cursor-pointer rounded-full border border-surface/40 px-4 py-2 font-sans text-caption font-bold text-surface"
      >
        {t('scan.switchToPick')}
      </button>
    </div>
  )
}

function PlaceList({
  visitId,
  defaultFloorId,
  onReported,
  onScanInstead,
}: {
  visitId: string
  defaultFloorId?: string
  onReported: () => void
  onScanInstead: () => void
}) {
  const t = useT()
  const { locale } = useLocale()
  const report = useReportLocation(visitId)
  const { data: servicePoints, isPending } = useServicePoints()

  // Places, not service points: several clinics share one nurse station, and
  // the report is about where the patient stands (one node per place).
  const places = new Map<string, { entryNodeId: string; name: string; floorCode: string }>()
  for (const sp of servicePoints ?? []) {
    const entryNodeId = sp.place?.entryNodeId
    if (!entryNodeId || !sp.place) continue
    if (!places.has(entryNodeId)) {
      places.set(entryNodeId, {
        entryNodeId,
        name: sp.place.name,
        floorCode: sp.place.floor.code,
      })
    }
  }

  // Floor-first: chips for the floors that have pickable places, ordered by
  // floor code, opening on the floor the map is already showing. The match
  // goes through the localized floor label — no floorId→code table here.
  const catalog = messagesFor(locale)
  const floorCodes = Array.from(new Set(Array.from(places.values()).map((p) => p.floorCode))).sort()
  const defaultFloorCode = defaultFloorId
    ? floorCodes.find(
        (code) =>
          floorLabelFor(defaultFloorId, locale) === format(catalog, 'common.floor', { code }),
      )
    : undefined
  const [activeFloor, setActiveFloor] = useState(defaultFloorCode ?? floorCodes[0] ?? '')
  const visible = Array.from(places.values()).filter((p) => p.floorCode === activeFloor)

  return (
    <div className="flex flex-1 flex-col gap-3 overflow-y-auto pt-4">
      {floorCodes.length > 1 && (
        <div className="flex gap-1.5">
          {floorCodes.map((code) => (
            <button
              key={code}
              type="button"
              onClick={() => setActiveFloor(code)}
              className={`cursor-pointer rounded-full border px-3.5 py-1.5 font-sans text-caption font-bold ${
                code === activeFloor
                  ? 'border-primary bg-primary-tint text-ink'
                  : 'border-surface/40 text-surface'
              }`}
            >
              {t('scan.floor', { floor: code })}
            </button>
          ))}
        </div>
      )}

      {isPending ? (
        <p className="m-0 font-sans text-body-sm text-surface">{t('scan.loadingPlaces')}</p>
      ) : visible.length === 0 ? (
        <p className="m-0 font-sans text-body-sm text-surface">{t('scan.noPlaces')}</p>
      ) : (
        visible.map((place) => (
          <button
            key={place.entryNodeId}
            type="button"
            disabled={report.isPending}
            onClick={() =>
              report.mutate(
                { source: 'MANUAL', raw: place.entryNodeId },
                { onSuccess: onReported },
              )
            }
            className="flex cursor-pointer items-baseline justify-between gap-3 rounded-lg border border-surface/40 bg-transparent px-4 py-3 text-left font-sans disabled:cursor-wait disabled:opacity-60"
          >
            <span className="font-sans text-body-sm font-bold text-surface">{place.name}</span>
            <span className="shrink-0 font-sans text-caption text-surface/70">
              {t('scan.floor', { floor: place.floorCode })}
            </span>
          </button>
        ))
      )}

      {report.error && (
        <p className="m-0 font-sans text-body-sm text-primary">
          {reportErrorLabel(report.error, t)}
        </p>
      )}

      <button
        type="button"
        onClick={onScanInstead}
        className="mt-auto cursor-pointer self-center rounded-full border border-surface/40 px-4 py-2 font-sans text-caption font-bold text-surface"
      >
        {t('scan.switchToCamera')}
      </button>
    </div>
  )
}

/** A patient-facing error line for a failed report, keyed off the status. */
function reportErrorLabel(error: ApiError, t: ReturnType<typeof useT>): string {
  // A QR the server cannot resolve is the client's input being wrong (400) —
  // phrased for the patient, not echoed from the API.
  if (error.status === 400) return t('scan.reportInvalid')
  return t('scan.reportFailed')
}
