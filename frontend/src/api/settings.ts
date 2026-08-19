import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { request, ApiError } from './client'
import type { Branding, BrandingResponse, SettingsResponse } from './types'

export const SETTINGS_KEY = ['settings']

export function useSettings() {
  return useQuery({
    queryKey: SETTINGS_KEY,
    queryFn: () => request<SettingsResponse>('/settings'),
  })
}

export function useBranding(enabled: boolean) {
  return useQuery({
    queryKey: ['branding'],
    queryFn: () => request<BrandingResponse>('/admin/branding'),
    enabled,
  })
}

type ActionState = 'idle' | 'pending' | 'success' | 'error'

/**
 * One settings action: send it, then report the server's own message next to the
 * control that triggered it.
 *
 * Every mutation refetches the settings payload rather than patching local state, so
 * what the screen shows is always what the server stored - a settings page that lies
 * about its own state is worse than one that takes an extra round trip.
 */
export function useSettingsAction<TInput, TResult extends { message?: string }>(
  send: (input: TInput) => Promise<TResult>,
  options: { invalidate?: unknown[][]; onSuccess?: (result: TResult) => void } = {},
) {
  const client = useQueryClient()

  const mutation = useMutation({
    mutationFn: send,
    onSuccess: async (result) => {
      const keys = options.invalidate ?? [SETTINGS_KEY, ['bootstrap']]
      await Promise.all(keys.map((key) => client.invalidateQueries({ queryKey: key })))
      options.onSuccess?.(result)
    },
  })

  const state: ActionState = mutation.isPending
    ? 'pending'
    : mutation.isError
      ? 'error'
      : mutation.isSuccess
        ? 'success'
        : 'idle'

  const message = mutation.isError
    ? (mutation.error as ApiError | Error).message
    : mutation.data?.message

  return { run: mutation.mutate, runAsync: mutation.mutateAsync, state, message, reset: mutation.reset }
}

type Message = { message: string }

export const settingsApi = {
  updateProfile: (body: {
    email: string
    location: string
    start_of_week: number
    reports_weekly: boolean
    public_leaderboard: boolean
  }) => request<Message>('/user/profile', { method: 'PATCH', body }),

  changePassword: (body: { password_old: string; password_new: string; password_repeat: string }) =>
    request<Message>('/user/password', { method: 'POST', body }),

  changeUsername: (body: { username: string }) =>
    request<Message & { relogin_required: boolean }>('/user/username', { method: 'POST', body }),

  resetApiKey: () =>
    request<Message & { api_key: string }>('/user/api-key/reset', { method: 'POST' }),

  updateSharing: (body: {
    max_days: number
    projects: boolean
    languages: boolean
    editors: boolean
    operating_systems: boolean
    machines: boolean
    labels: boolean
    activity_chart: boolean
  }) => request<Message>('/user/sharing', { method: 'PUT', body }),

  updateLeaderboardOptIn: (body: { enabled: boolean }) =>
    request<Message>('/user/leaderboard-opt-in', { method: 'PUT', body }),

  updateExcludeUnknownProjects: (body: { enabled: boolean }) =>
    request<Message>('/user/exclude-unknown-projects', { method: 'PUT', body }),

  updateHeartbeatsTimeout: (body: { minutes: number }) =>
    request<Message>('/user/heartbeats-timeout', { method: 'PUT', body }),

  updateReadmeStatsBaseUrl: (body: { base_url: string }) =>
    request<Message>('/user/readme-stats-base-url', { method: 'PUT', body }),

  deleteAccount: () => request<Message>('/user', { method: 'DELETE' }),

  createAlias: (body: { key: string; value: string; type: number }) =>
    request<Message>('/aliases', { method: 'POST', body }),

  deleteAlias: (key: string, type: number) =>
    request<Message>('/aliases', { method: 'DELETE', params: { key, type: String(type) } }),

  createLabels: (body: { label: string; projects: string[] }) =>
    request<Message>('/labels', { method: 'POST', body }),

  deleteLabel: (label: string, project: string) =>
    request<Message>('/labels', { method: 'DELETE', params: { label, project } }),

  createLanguageMapping: (body: { extension: string; language: string }) =>
    request<Message>('/language-mappings', { method: 'POST', body }),

  deleteLanguageMapping: (id: number) =>
    request<Message>(`/language-mappings/${id}`, { method: 'DELETE' }),

  createApiKey: (body: { label: string; read_only: boolean }) =>
    request<{ label: string; api_key: string; read_only: boolean }>('/api-keys', {
      method: 'POST',
      body,
    }),

  deleteApiKey: (key: string) =>
    request<Message>(`/api-keys/${encodeURIComponent(key)}`, { method: 'DELETE' }),

  updateWakatime: (body: { api_key: string; api_url: string }) =>
    request<Message>('/integrations/wakatime', { method: 'PUT', body }),

  importWakatime: (body: { use_legacy_importer: boolean }) =>
    request<Message>('/integrations/wakatime/import', { method: 'POST', body }),

  regenerateSummaries: () => request<Message>('/data/regenerate', { method: 'POST' }),

  clearData: () => request<Message>('/data/clear', { method: 'POST' }),

  createInvite: () => request<{ code: string; link: string }>('/invites', { method: 'POST' }),

  webauthnRegistrationOptions: () => request<unknown>('/webauthn/registration-options'),

  createWebauthnCredential: (body: { name: string; credential: string }) =>
    request<Message>('/webauthn/credentials', { method: 'POST', body }),

  deleteWebauthnCredential: (name: string) =>
    request<Message>(`/webauthn/credentials/${encodeURIComponent(name)}`, { method: 'DELETE' }),

  updateBranding: (body: Branding) =>
    request<Message & { rejected: string[]; effective: Branding }>('/admin/branding', {
      method: 'PUT',
      body,
    }),
}
