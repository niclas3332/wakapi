import { serverConfig } from './bootstrap'

/**
 * Cookie the server has always used to remember the last chosen range across
 * visits (models.PersistentIntervalKey).
 */
const INTERVAL_COOKIE = 'wakapi_summary_interval'

export const DEFAULT_INTERVAL = 'today'

export type IntervalOption = {
  value: string
  label: string
  /** Single-key shortcut, carried over from the legacy time picker. */
  hotkey: string
}

/** Same options and shortcuts the legacy time picker offered. */
export const INTERVAL_OPTIONS: IntervalOption[] = [
  { value: 'today', label: 'Today', hotkey: 'd' },
  { value: 'yesterday', label: 'Yesterday', hotkey: 'e' },
  { value: 'week', label: 'This Week', hotkey: 'w' },
  { value: 'month', label: 'This Month', hotkey: 'm' },
  { value: 'year', label: 'This Year', hotkey: 'y' },
  { value: 'last_7_days', label: 'Past 7 Days', hotkey: '1' },
  { value: 'last_30_days', label: 'Past 30 Days', hotkey: '2' },
  { value: 'last_6_months', label: 'Past 6 Months', hotkey: '3' },
  { value: 'last_12_months', label: 'Past 12 Months', hotkey: '4' },
  { value: 'any', label: 'All Time', hotkey: 'a' },
]

export function intervalLabel(interval: string, from?: string, to?: string): string {
  if (from && to) return `${from} — ${to}`
  return INTERVAL_OPTIONS.find((o) => o.value === interval)?.label ?? interval
}

export function readIntervalCookie(): string | null {
  const match = document.cookie.match(new RegExp(`(?:^|; )${INTERVAL_COOKIE}=([^;]*)`))
  return match ? decodeURIComponent(match[1]) : null
}

/**
 * Persist the chosen range.
 *
 * Path must be set explicitly: the legacy server-side write emitted a bare
 * Set-Cookie, which browsers scope to /summary. Writing from any other route
 * without a path would create a second cookie that shadows it.
 */
export function writeIntervalCookie(interval: string): void {
  const path = serverConfig.basePath || '/'
  document.cookie = `${INTERVAL_COOKIE}=${encodeURIComponent(interval)}; path=${path}; SameSite=Lax; max-age=${60 * 60 * 24 * 365}`
}

/** Filters the summary endpoints accept, in the order the UI shows them. */
export const FILTER_KEYS = ['project', 'language', 'editor', 'machine', 'operating_system', 'label', 'category', 'branch', 'entity'] as const

export type FilterKey = (typeof FILTER_KEYS)[number]

export type RangeParams = {
  interval?: string
  from?: string
  to?: string
}

/**
 * Turn the current url search params into the query the api expects.
 *
 * The url is the source of truth for range and filters, which is what keeps deep
 * links and browser back/forward working the way they did before.
 */
export function toApiParams(search: URLSearchParams): Record<string, string> {
  const params: Record<string, string> = {}

  const from = search.get('from')
  const to = search.get('to')
  if (from && to) {
    params.from = from
    params.to = to
  } else {
    params.interval = search.get('interval') || readIntervalCookie() || DEFAULT_INTERVAL
  }

  for (const key of FILTER_KEYS) {
    const value = search.get(key)
    if (value) params[key] = value
  }
  return params
}

export function currentInterval(search: URLSearchParams): string {
  if (search.get('from') && search.get('to')) return ''
  return search.get('interval') || readIntervalCookie() || DEFAULT_INTERVAL
}

export function activeFilters(search: URLSearchParams): { key: FilterKey; value: string }[] {
  return FILTER_KEYS.flatMap((key) => {
    const value = search.get(key)
    return value ? [{ key, value }] : []
  })
}
