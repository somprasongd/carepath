/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** CarePath API base URL; defaults to the local docker-compose api on :8080. */
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
