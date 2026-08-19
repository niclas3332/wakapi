import { useQuery } from '@tanstack/react-query'

import { request } from './client'
import { normalizeSummaryResponse } from './normalize'
import type {
  AIFilesResponse,
  AIOverviewResponse,
  BootstrapResponse,
  HourlyResponse,
  InsightsResponse,
  LeaderboardResponse,
  Paged,
  ProjectItem,
  SummaryResponse,
  TimelineResponse,
  WrappedResponse,
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

export function useInsights(params: Record<string, string>, enabled = true) {
  return useQuery({
    queryKey: ['insights', params],
    enabled,
    queryFn: async ({ signal }) => {
      const data = await request<InsightsResponse>('/insights', { params, signal })
      // Go marshals a nil slice as null, and an empty range has nothing in any of these.
      return {
        ...data,
        daily: data.daily ?? [],
        weekdays: data.weekdays ?? [],
        projects: data.projects ?? [],
        languages: data.languages ?? [],
        editors: data.editors ?? [],
        operating_systems: data.operating_systems ?? [],
        machines: data.machines ?? [],
        categories: data.categories ?? [],
      }
    },
  })
}

export function useWrappedYears() {
  return useQuery({
    queryKey: ['wrapped-years'],
    queryFn: async () => {
      const data = await request<{ years: number[] }>('/wrapped/years')
      return data.years ?? []
    },
    staleTime: 60 * 60 * 1000,
  })
}

export function useWrapped(year: number | undefined) {
  return useQuery({
    queryKey: ['wrapped', year],
    queryFn: async ({ signal }) => {
      const data = await request<WrappedResponse>(`/wrapped/${year}`, { signal })
      return { ...data, daily: data.daily ?? [], weekdays: data.weekdays ?? [] }
    },
    enabled: year !== undefined,
  })
}

export function useAIOverview(params: Record<string, string>) {
  return useQuery({
    queryKey: ['ai-overview', params],
    queryFn: async ({ signal }) => {
      const data = await request<AIOverviewResponse>('/ai/overview', { params, signal })
      return {
        ...data,
        by_model: data.by_model ?? [],
        by_day: data.by_day ?? [],
        by_project: data.by_project ?? [],
      }
    },
  })
}

export function useAIFiles(params: Record<string, string>, kind: 'ai' | 'human') {
  return useQuery({
    queryKey: ['ai-files', params, kind],
    queryFn: async ({ signal }) => {
      const data = await request<AIFilesResponse>('/ai/files', {
        params: { ...params, kind },
        signal,
      })
      return { ...data, items: data.items ?? [] }
    },
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
