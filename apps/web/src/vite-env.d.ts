/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** API base override; unset/empty = same-origin (dev Vite proxy, prod nginx edge). */
  readonly VITE_API_BASE_URL?: string
  /** 'line' uses real LIFF login; anything else falls back to the demo stub. */
  readonly VITE_AUTH_MODE?: 'line' | 'demo'
  /** LIFF app ID from the LINE Developers console; required when VITE_AUTH_MODE=line. */
  readonly VITE_LIFF_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
