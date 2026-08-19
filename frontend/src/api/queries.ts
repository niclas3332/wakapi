import { useQuery } from '@tanstack/react-query'

import { request } from './client'
import { normalizeSummaryResponse } from './normalize'
import type {
  BootstrapResponse,
  HourlyResponse,
  LeaderboardResponse,
  Paged,
  ProjectItem,
  SummaryResponse,
  TimelineResponse,
} from './types'

/** The signed-in user changes rarely; settings mutations invalidate this key. */
export function useBootstrap() {
  return useQuery({
    queryKey: ['bootstrap'],
    queryFn: () => request<BootstrapResponse>('/bootstrap'),
    staleTime: Infinity,
  })
}

export function useSummary(params: Record<string, string>) {
  return useQuery({
    queryKey: ['summary', params],
    queryFn: async ({ signal }) =>
      normalizeSummaryResponse(await request<SummaryResponse>('/summary', { params, signal })),
  })
}

export function useTimeline(params: Record<string, string>) {
  return useQuery({
    queryKey: ['timeline', params],
    queryFn: async ({ signal }) => {
      const data = await request<TimelineResponse>('/summary/timeline', { params, signal })
      return { ...data, items: data.items ?? [] }
    },
  })
}

export function useHourly(params: Record<string, string>) {
  return useQuery({
    queryKey: ['hourly', params],
    queryFn: async ({ signal }) => {
      const data = await request<HourlyResponse>('/summary/hourly', { params, signal })
      return { ...data, items: (data.items ?? []).map((p) => ({ ...p, items: p.items ?? [] })) }
    },
  })
}

export function useProjects(page: number, pageSize: number, query: string) {
  return useQuery({
    queryKey: ['projects', page, pageSize, query],
    queryFn: ({ signal }) =>
      request<Paged<ProjectItem>>('/projects', {
        // page and page_size must travel together: the server only honours a page
        // size when a page number is present too.
        params: { page: String(page), page_size: String(pageSize), ...(query ? { q: query } : {}) },
        signal,
      }),
    placeholderData: (previous) => previous,
  })
}

export function useLeaderboard(by: string, key: string) {
  return useQuery({
    queryKey: ['leaderboard', by, key],
    queryFn: async ({ signal }) => {
      const data = await request<LeaderboardResponse>('/leaderboard', {
        params: { page: '1', page_size: '100', ...(by ? { by } : {}), ...(key ? { key } : {}) },
        signal,
      })
      // top_keys is nil unless an aggregation is active, and Go marshals nil as null.
      return { ...data, items: data.items ?? [], top_keys: data.top_keys ?? [] }
    },
  })
}
