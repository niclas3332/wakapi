/**
 * Mirrors of the DTOs in routes/api/ui. Kept hand-written rather than generated so
 * the shape stays readable; the Go structs are the source of truth.
 */

/** Every duration in this api is a plain number of seconds. */
export type SummaryItem = {
  key: string
  seconds: number
}

export type Summary = {
  from: string
  to: string
  num_heartbeats: number
  projects: SummaryItem[]
  languages: SummaryItem[]
  editors: SummaryItem[]
  operating_systems: SummaryItem[]
  machines: SummaryItem[]
  labels: SummaryItem[]
  branches: SummaryItem[]
  entities: SummaryItem[]
  categories: SummaryItem[]
}

export type AvailableFilters = {
  projects: string[]
  languages: string[]
  machines: string[]
  labels: string[]
  categories: string[]
}

export type SummaryResponse = {
  summary: Summary
  available_filters: AvailableFilters
  editor_colors: Record<string, string>
  language_colors: Record<string, string>
  os_colors: Record<string, string>
  user_first_data: string
  data_retention_months: number
}

export type TimelineItem = {
  date: string
  projects: { name: string; seconds: number }[]
}

export type TimelineResponse = {
  items: TimelineItem[]
  available: boolean
  min_range_days: number
  max_range_days: number
}

export type HourlyBreakdownItem = {
  from_time: string
  seconds: number
  entity: string
}

export type HourlyProject = {
  project: string
  items: HourlyBreakdownItem[]
}

export type HourlyResponse = {
  from: string
  items: HourlyProject[]
  truncated: boolean
  max_durations: number
}

export type ProjectItem = {
  project: string
  top_language: string
  /** Heartbeats recorded. Kept for tooltips; the list shows `seconds`. */
  count: number
  /** All-time total tracked on this project. */
  seconds: number
  first: string
  last: string
}

export type Paged<T> = {
  items: T[]
  page: number
  page_size: number
  has_more: boolean
}

export type LeaderboardItem = {
  rank: number
  user_id: string
  seconds: number
  key: string | null
  is_me: boolean
  languages?: string[]
}

export type LeaderboardResponse = {
  items: LeaderboardItem[]
  by: string
  key: string
  top_keys: string[]
  interval_label: string
  page: number
  page_size: number
  has_more: boolean
}

export type Sharing = {
  max_days: number
  projects: boolean
  languages: boolean
  editors: boolean
  operating_systems: boolean
  machines: boolean
  labels: boolean
  activity_chart: boolean
}

export type User = {
  id: string
  email: string
  api_key: string
  location: string
  start_of_week: number
  created_at: string
  last_logged_in_at: string
  auth_type: string
  is_admin: boolean
  has_data: boolean
  reports_weekly: boolean
  public_leaderboard: boolean
  exclude_unknown_projects: boolean
  heartbeats_timeout_sec: number
  readme_stats_base_url: string
  sharing: Sharing
  wakatime_connected: boolean
  wakatime_api_url: string
  has_active_subscription: boolean
  subscribed_until: string | null
}

export type BootstrapResponse = {
  user: User
}
