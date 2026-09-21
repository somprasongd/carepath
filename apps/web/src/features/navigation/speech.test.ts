import { afterEach, describe, expect, it } from 'vitest'
import { cancelSpeech, speakCues, speechSupported } from './speech'

/**
 * The Web Speech API surface (#108) tested against a stubbed window —
 * vitest runs in node, where no real engine exists, and that is itself the
 * first case: where the browser cannot speak, these functions must stay
 * quiet (return false / no-op), never throw. Where it can (the stub), the
 * contract is: cancel whatever was being said, then queue one utterance per
 * cue in the active locale's BCP-47 tag, preferring a device voice that
 * speaks that language.
 */

type Spoken = { text: string; lang: string; voiceURI: string | undefined }

type SpeechStub = {
  spoken: Spoken[]
  cancelled: () => number
  setVoices: (voices: { lang: string; voiceURI: string }[]) => void
  makeGetVoicesThrow: () => void
  restore: () => void
}

function installSpeechStub(): SpeechStub {
  const spoken: Spoken[] = []
  let cancelledCount = 0
  let voices: { lang: string; voiceURI: string }[] = []
  let getVoicesThrows = false
  const synth = {
    cancel() {
      cancelledCount++
    },
    speak(utterance: { text: string; lang: string; voice?: { voiceURI: string } }) {
      spoken.push({ text: utterance.text, lang: utterance.lang, voiceURI: utterance.voice?.voiceURI })
    },
    getVoices() {
      if (getVoicesThrows) throw new Error('voices not ready')
      return voices
    },
  }
  const previous = (globalThis as { window?: unknown }).window
  ;(globalThis as { window?: unknown }).window = {
    speechSynthesis: synth,
    SpeechSynthesisUtterance: class {
      text: string
      lang = ''
      voice?: { voiceURI: string }
      constructor(text: string) {
        this.text = text
      }
    },
  }
  return {
    spoken,
    cancelled: () => cancelledCount,
    setVoices(next) {
      voices = next
    },
    makeGetVoicesThrow() {
      getVoicesThrows = true
    },
    restore() {
      ;(globalThis as { window?: unknown }).window = previous
    },
  }
}

let cleanup: SpeechStub | null = null
afterEach(() => {
  cleanup?.restore()
  cleanup = null
})

describe('speechSupported', () => {
  it('is false without a speech engine (node, or a WebView without one)', () => {
    expect(speechSupported()).toBe(false)
  })

  it('is true when the browser exposes speechSynthesis', () => {
    cleanup = installSpeechStub()
    expect(speechSupported()).toBe(true)
  })
})

describe('speakCues', () => {
  it('returns false and stays silent where the browser cannot speak', () => {
    expect(speakCues(['a cue'], 'th')).toBe(false)
  })

  it('cancels what was playing, then queues one utterance per cue in Thai', () => {
    cleanup = installSpeechStub()
    const stub = cleanup
    expect(speakCues(['first', 'second'], 'th')).toBe(true)
    expect(stub.cancelled()).toBe(1)
    expect(stub.spoken).toEqual([
      { text: 'first', lang: 'th-TH', voiceURI: undefined },
      { text: 'second', lang: 'th-TH', voiceURI: undefined },
    ])
  })

  it('speaks English cues as en-US and prefers a matching device voice', () => {
    cleanup = installSpeechStub()
    const stub = cleanup
    stub.setVoices([
      { lang: 'th-TH', voiceURI: 'th-voice' },
      { lang: 'en_US', voiceURI: 'en-voice' },
    ])
    speakCues(['go'], 'en')
    // The underscore form matches too — devices report either shape.
    expect(stub.spoken).toEqual([{ text: 'go', lang: 'en-US', voiceURI: 'en-voice' }])
  })

  it('falls back to the default voice when getVoices fails', () => {
    cleanup = installSpeechStub()
    const stub = cleanup
    stub.makeGetVoicesThrow()
    expect(speakCues(['go'], 'th')).toBe(true)
    expect(stub.spoken).toEqual([{ text: 'go', lang: 'th-TH', voiceURI: undefined }])
  })

  it('queues nothing for an empty cue set', () => {
    cleanup = installSpeechStub()
    const stub = cleanup
    expect(speakCues([], 'th')).toBe(false)
    expect(stub.spoken).toEqual([])
    expect(stub.cancelled()).toBe(0)
  })
})

describe('cancelSpeech', () => {
  it('cancels through the engine', () => {
    cleanup = installSpeechStub()
    const stub = cleanup
    cancelSpeech()
    expect(stub.cancelled()).toBe(1)
  })

  it('is a no-op where the browser cannot speak', () => {
    expect(() => cancelSpeech()).not.toThrow()
  })
})
