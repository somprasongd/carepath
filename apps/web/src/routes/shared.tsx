import { createFileRoute } from '@tanstack/react-router'
import { SharedScreen } from './-SharedScreen'

/**
 * ญาติ · ติดตามการรักษา (#90) — deliberately outside /patient and /staff:
 * no AuthProvider, no LIFF, no menu. The only credential this surface ever
 * holds is the share token in the URL fragment (ADR-0011 §5), and the only
 * data it can read is the one redacted journey that token unlocks.
 */
export const Route = createFileRoute('/shared')({
  component: SharedScreen,
})
