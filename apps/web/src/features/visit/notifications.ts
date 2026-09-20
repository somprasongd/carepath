import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPut } from '@/api/client'
import type { components } from '@/api/schema'

export type VisitNotificationPref = components['schemas']['VisitNotificationPref']

/**
 * The visit's queue-proximity notification toggle (FR-21, #104): read once
 * per visit — the preference changes only when this patient flips it, so
 * there is nothing to poll. Default-on comes from the API (absent row means
 * enabled), not a client-side fallback.
 */
export function useVisitNotificationPref(visitId: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['visit', visitId, 'notification-pref'] as const,
    queryFn: () =>
      apiGet<VisitNotificationPref>(
        `/api/v1/journeys/${encodeURIComponent(visitId)}/notifications`,
      ),
    enabled: options?.enabled ?? true,
  })
}

/**
 * Flip the toggle. The mutation writes the pref cache directly so the
 * switch reflects the choice immediately — the sweep picks it up on its
 * next pass server-side.
 */
export function useSetVisitNotificationPref(visitId: string) {
  const queryClient = useQueryClient()
  const key = ['visit', visitId, 'notification-pref'] as const

  return useMutation({
    mutationFn: (enabled: boolean) =>
      apiPut<VisitNotificationPref>(
        `/api/v1/journeys/${encodeURIComponent(visitId)}/notifications`,
        { enabled },
      ),
    onSuccess: (pref) => {
      queryClient.setQueryData(key, pref)
    },
  })
}
