import { queryOptions, useQuery } from '@tanstack/react-query'
import { apiGet } from '@/api/client'
import type { components } from '@/api/schema'

export type ServicePoint = components['schemas']['ServicePoint']

/**
 * The service → destination mapping (#24): every active service point with
 * its resolved place and floor. Read-only for the MVP (FR-11) — the patient
 * destination lookup and the staff console share this one source.
 */
export function servicePointsQueryOptions() {
  return queryOptions({
    queryKey: ['service-points'] as const,
    queryFn: () => apiGet<ServicePoint[]>('/api/v1/service-points'),
    staleTime: 60_000,
  })
}

export function useServicePoints() {
  return useQuery(servicePointsQueryOptions())
}
