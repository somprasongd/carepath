import { queryOptions, useQuery } from '@tanstack/react-query'
import { apiGet } from '@/api/client'
import type { components } from '@/api/schema'

export type AnalyticsOverview = components['schemas']['AnalyticsOverview']
export type AnalyticsSummary = components['schemas']['AnalyticsSummary']
export type ServicePointMetrics = components['schemas']['ServicePointMetrics']

/**
 * The executive dashboard's single source (#87): the aggregate #86 serves.
 * Polls every 15s — the same cadence as the staff queue screens — so a
 * console left open on a wall display tracks the day without a reload.
 */
export function analyticsOverviewQueryOptions() {
  return queryOptions({
    queryKey: ['analytics', 'overview'] as const,
    queryFn: () => apiGet<AnalyticsOverview>('/api/v1/analytics/overview?window=today'),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useAnalyticsOverview() {
  return useQuery(analyticsOverviewQueryOptions())
}
