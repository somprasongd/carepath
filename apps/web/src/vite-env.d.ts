/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** CarePath API base URL; defaults to the local docker-compose api on :8080. */
  readonly VITE_API_BASE_URL?: string
  /** 'line' uses real LIFF login; anything else falls back to the demo stub. */
  readonly VITE_AUTH_MODE?: 'line' | 'demo'
  /** LIFF app ID from the LINE Developers console; required when VITE_AUTH_MODE=line. */
  readonly VITE_LIFF_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
