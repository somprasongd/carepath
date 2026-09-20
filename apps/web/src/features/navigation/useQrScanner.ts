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
  const lastDetected = useRef<string | null>(null)
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
      while (!cancelled) {
        try {
          const codes = await detector.detect(video)
          const raw = codes.find((code) => code.rawValue !== '')?.rawValue
          if (raw && raw !== lastDetected.current) {
            lastDetected.current = raw
            detectRef.current(raw)
          }
        } catch {
          // A frame too early (readyState < HAVE_METADATA) throws; retry on
          // the next tick rather than killing the loop.
        }
        await new Promise((resolve) => setTimeout(resolve, 250))
      }
    })()

    return stop
  }, [])

  return { videoRef, state }
}
