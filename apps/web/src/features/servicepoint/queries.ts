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

/**
 * The signed-in staff member's assigned points (FR-15, #102) — the
 * workstation picker feed. Assignment, not existence, decides what a staff
 * member sees; admins get the full list so the picker doubles as their way
 * to watch any station.
 */
export function myServicePointsQueryOptions() {
  return queryOptions({
    queryKey: ['staff', 'my-service-points'] as const,
    queryFn: () => apiGet<ServicePoint[]>('/api/v1/staff/my/service-points'),
    staleTime: 60_000,
  })
}

export function useMyServicePoints() {
  return useQuery(myServicePointsQueryOptions())
}
