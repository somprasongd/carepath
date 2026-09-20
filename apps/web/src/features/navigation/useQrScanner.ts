import { useEffect, useRef, useState } from 'react'

export type QrScannerState =
  | 'scanning'
  | 'unsupported'
  | { denied: string }

/**
 * Whether this browser can drive the camera QR scanner at all: the native
 * BarcodeDetector (Chromium, Safari 17+) behind a camera permission. Firefox
 * and older WebViews report unsupported — the overlay then routes to the
 * manual place list instead of a dead camera view.
 */
export function qrScannerSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    'BarcodeDetector' in window &&
    navigator.mediaDevices?.getUserMedia != null
  )
}

/**
 * The detection loop, extracted from the hook so its contract is testable
 * without a camera: one callback per *distinct* raw value (a sticker held in
 * view fires the detector continuously — the report must not queue per
 * frame), a throwing frame (video not warmed up yet) skips to the next tick
 * instead of killing the loop, and a flipped `isCancelled` ends it.
 */
export async function runQrDetectionLoop(options: {
  detectFrame: () => Promise<string | undefined>
  onDetect: (raw: string) => void
  isCancelled: () => boolean
  intervalMs?: number
}): Promise<void> {
  const { detectFrame, onDetect, isCancelled, intervalMs = 250 } = options
  let lastDetected: string | null = null
  while (!isCancelled()) {
    try {
      const raw = await detectFrame()
      if (raw && raw !== lastDetected) {
        lastDetected = raw
        onDetect(raw)
      }
    } catch {
      // A frame too early (readyState < HAVE_METADATA) throws; retry on
      // the next tick rather than killing the loop.
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs))
  }
}

/**
 * Live QR scanning for the navigate screen's location report: opens the back
 * camera into `videoRef`, detects in a loop, and calls `onDetect` once per
 * *new* raw value (a sticker held in view fires repeatedly; the report must
 * not). `unsupported` is decided once at mount from the environment; a denied
 * permission lands as its own state after the user answers the prompt. Camera
 * teardown happens in the effect's cleanup — unmounting the overlay is what
 * stops the stream.
 */
export function useQrScanner(onDetect: (raw: string) => void) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [state, setState] = useState<QrScannerState>(() =>
    qrScannerSupported() ? 'scanning' : 'unsupported',
  )
  const detectRef = useRef(onDetect)

  // Keep the latest callback reachable from the detection loop without
  // restarting the camera on every parent render.
  useEffect(() => {
    detectRef.current = onDetect
  })

  useEffect(() => {
    if (!qrScannerSupported()) return

    let cancelled = false
    let stream: MediaStream | undefined

    const stop = () => {
      cancelled = true
      stream?.getTracks().forEach((track) => track.stop())
      if (videoRef.current) videoRef.current.srcObject = null
    }

    void (async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: 'environment' } },
          audio: false,
        })
      } catch (cause) {
        if (!cancelled) setState({ denied: String(cause) })
        return
      }
      if (cancelled) {
        stream.getTracks().forEach((track) => track.stop())
        return
      }

      const video = videoRef.current
      if (!video) {
        stream.getTracks().forEach((track) => track.stop())
        return
      }
      video.srcObject = stream
      try {
        await video.play()
      } catch {
        // Autoplay refusal on a muted inline video is unexpected, not fatal —
        // detection reads the element whether or not it paints frames.
      }

      const detector = new BarcodeDetector({ formats: ['qr_code'] })
      await runQrDetectionLoop({
        detectFrame: async () => {
          const codes = await detector.detect(video)
          return codes.find((code) => code.rawValue !== '')?.rawValue
        },
        onDetect: (raw) => detectRef.current(raw),
        isCancelled: () => cancelled,
      })
    })()

    return stop
  }, [])

  return { videoRef, state }
}
