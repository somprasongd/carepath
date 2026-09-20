import { describe, expect, it } from 'vitest'
import { qrScannerSupported, runQrDetectionLoop } from './useQrScanner'

// Real QR payloads the loop must dedupe on (apps/api/internal/location/qr:
// a place URL, or a bare node reference) — a sticker held in view re-fires
// the detector with the same string every tick.
const PLACE_QR = 'https://carepath.example/location/REG-01'
const NODE_QR = 'node/I-1301/node-reception'

/** Drives the loop with a scripted sequence of frames, then cancels it. */
async function runFrames(frames: Array<string | undefined | Error>) {
  const detected: string[] = []
  let tick = 0
  let cancelled = false
  const loop = runQrDetectionLoop({
    detectFrame: async () => {
      const frame = frames[Math.min(tick, frames.length - 1)]
      tick += 1
      if (frame instanceof Error) throw frame
      return frame
    },
    onDetect: (raw) => detected.push(raw),
    isCancelled: () => cancelled,
    intervalMs: 0,
  })
  // One macrotask advances one loop iteration (the intervalMs:0 sleep);
  // extra rounds are harmless — the script holds its last frame and the
  // dedupe keeps them from firing again.
  for (let i = 0; i < frames.length + 2 && !cancelled; i++) {
    await new Promise((resolve) => setTimeout(resolve, 0))
  }
  cancelled = true
  await loop
  return { detected, framesRead: tick }
}

describe('runQrDetectionLoop (#98 scanner semantics)', () => {
  it('fires once per distinct raw value — a held sticker must not queue reports', async () => {
    const { detected } = await runFrames([PLACE_QR, PLACE_QR, PLACE_QR, NODE_QR, NODE_QR])
    expect(detected).toEqual([PLACE_QR, NODE_QR])
  })

  it('skips empty frames without resetting the last detection', async () => {
    // The same code re-appearing after blank frames is still the same
    // sticker — it must not report again.
    const { detected } = await runFrames([PLACE_QR, undefined, undefined, PLACE_QR])
    expect(detected).toEqual([PLACE_QR])
  })

  it('survives a throwing frame (video not warmed up) and detects on the next tick', async () => {
    const { detected } = await runFrames([new Error('not started'), PLACE_QR, PLACE_QR])
    expect(detected).toEqual([PLACE_QR])
  })

  it('stops reading frames once cancelled', async () => {
    let cancelled = false
    let reads = 0
    const loop = runQrDetectionLoop({
      detectFrame: async () => {
        reads += 1
        return PLACE_QR
      },
      onDetect: () => {},
      isCancelled: () => cancelled,
      intervalMs: 0,
    })
    await new Promise((resolve) => setTimeout(resolve, 2))
    cancelled = true
    await loop
    const settled = reads
    await new Promise((resolve) => setTimeout(resolve, 2))
    expect(reads).toBe(settled)
  })
})

describe('qrScannerSupported (#98 fallback gate)', () => {
  it('is false where BarcodeDetector or getUserMedia is missing (Firefox, old WebViews)', () => {
    // The node test environment has neither window nor navigator — the same
    // shape as a bare WebView — and the gate must route those to the manual
    // place list instead of a dead camera view.
    expect(qrScannerSupported()).toBe(false)
  })
})
