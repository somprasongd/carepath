import { useEffect, useRef, useState } from 'react'
import type { Locale } from '@/i18n'

/**
 * Voice guidance (#108, FR-25): speaking the navigate screen's turn cues
 * through the browser's own Web Speech API — no server TTS, no added
 * dependency; the voice (and its Thai quality) is whatever the device has.
 *
 * The privacy rule is structural: the only strings that may reach
 * `speakCues` come from `voiceSteps`, which builds them from catalog text
 * and floor labels alone and has no parameter through which a service
 * point, clinic, patient, or visit identity could enter. Nothing in this
 * file reads journey or visit data.
 *
 * Browsers that cannot speak (some in-app WebViews, e.g. iOS WKWebView)
 * report unsupported and the screen hides the toggle — never a dead button
 * (DESIGN.md's no-fake-capability rule).
 */

/** BCP-47 tag per app locale, for utterance.lang and voice matching. */
const SPEECH_LANG: Record<Locale, string> = { th: 'th-TH', en: 'en-US' }

/** Whether this browser can speak at all; false hides the voice toggle. */
export function speechSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    'speechSynthesis' in window &&
    typeof window.SpeechSynthesisUtterance === 'function'
  )
}

/** Best-effort device voice for the language; undefined keeps the default. */
function matchingVoice(synth: SpeechSynthesis, lang: string): SpeechSynthesisVoice | undefined {
  let voices: SpeechSynthesisVoice[]
  try {
    voices = synth.getVoices()
  } catch {
    return undefined
  }
  const wanted = lang.toLowerCase()
  return (
    voices.find((voice) => voice.lang.toLowerCase().replace('_', '-').startsWith(wanted)) ??
    voices.find((voice) => voice.lang.toLowerCase().slice(0, 2) === wanted.slice(0, 2))
  )
}

/**
 * Queue the cues as speech in the active locale, replacing whatever was
 * being said. One utterance per cue keeps the engine's pacing between
 * steps. Returns whether anything was actually queued (unsupported browser
 * or empty cues → false, silently — a missing voice is never an error the
 * patient should see).
 */
export function speakCues(cues: readonly string[], locale: Locale): boolean {
  if (!speechSupported() || cues.length === 0) return false
  const synth = window.speechSynthesis
  synth.cancel()
  const lang = SPEECH_LANG[locale]
  const voice = matchingVoice(synth, lang)
  for (const cue of cues) {
    const utterance = new window.SpeechSynthesisUtterance(cue)
    utterance.lang = lang
    if (voice) utterance.voice = voice
    synth.speak(utterance)
  }
  return true
}

/** Stop any in-flight announcement: disable, cue swap, or leaving the screen. */
export function cancelSpeech(): void {
  if (speechSupported()) window.speechSynthesis.cancel()
}

function keyOf(cues: readonly string[]): string {
  return cues.join('\u0000')
}

export type VoiceGuidance = {
  supported: boolean
  enabled: boolean
  toggle: () => void
}

/**
 * The navigate screen's voice switch (#108). Opt-in per session — NOT a
 * stored preference like accessibleOnly (#99): the autoplay policy means
 * speech must start from a user gesture, so a remembered "on" would render
 * a toggle that lies (pressed, but silent) after a reload, and a voice that
 * starts unprompted in a public corridor is wrong anyway.
 *
 * Enabling speaks inside the click handler itself — iOS Safari only unlocks
 * speechSynthesis within a user-gesture call stack. While on, a changed cue
 * set (a fresh location fix redrawing the route, a locale switch) re-speaks
 * via the effect; identical content is skipped so the enabling speak is not
 * doubled. Disabling or unmounting cancels immediately.
 */
export function useVoiceGuidance(cues: readonly string[], locale: Locale): VoiceGuidance {
  const [enabled, setEnabled] = useState(false)
  const lastSpokenRef = useRef<string | null>(null)

  const toggle = () => {
    const next = !enabled
    setEnabled(next)
    if (next) {
      speakCues(cues, locale)
      lastSpokenRef.current = keyOf(cues)
    } else {
      cancelSpeech()
      lastSpokenRef.current = null
    }
  }

  useEffect(() => {
    if (!enabled) return
    const key = keyOf(cues)
    if (lastSpokenRef.current === key) return
    lastSpokenRef.current = key
    speakCues(cues, locale)
  }, [enabled, cues, locale])

  // Speech outlives navigation on some engines — leaving the screen silences it.
  useEffect(() => cancelSpeech, [])

  return { supported: speechSupported(), enabled, toggle }
}
