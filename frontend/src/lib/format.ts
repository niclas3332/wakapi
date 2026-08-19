/**
 * Every duration from /api/ui is a plain number of seconds. The server converts at
 * the edge, because the Go models are not consistent about it - summary item totals
 * count seconds while duration and leaderboard values are nanoseconds.
 */

/** "4 hrs 12 mins", matching the wording the legacy pages used. */
export function formatDuration(seconds: number): string {
  const totalMinutes = Math.floor(seconds / 60)
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (hours === 0 && minutes === 0) {
    const rounded = Math.round(seconds)
    return rounded > 0 ? `${rounded} secs` : '0 mins'
  }
  if (hours === 0) return `${minutes} ${minutes === 1 ? 'min' : 'mins'}`
  if (minutes === 0) return `${hours} ${hours === 1 ? 'hr' : 'hrs'}`
  return `${hours} ${hours === 1 ? 'hr' : 'hrs'} ${minutes} ${minutes === 1 ? 'min' : 'mins'}`
}

/**
 * "8h 58m" - for stat tiles, where the long form does not fit the column and gets
 * truncated. Prose form stays in lists and tooltips, which have the room.
 */
export function formatDurationCompact(seconds: number): string {
  const totalMinutes = Math.floor(seconds / 60)
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (hours === 0 && minutes === 0) return `${Math.round(seconds)}s`
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

/** "4:12" - compact form for table cells and chart labels. */
export function formatDurationShort(seconds: number): string {
  const totalMinutes = Math.floor(seconds / 60)
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return `${hours}:${String(minutes).padStart(2, '0')}`
}

export function formatPercent(value: number, fractionDigits = 0): string {
  return `${(value * 100).toFixed(fractionDigits)}%`
}

const compactNumber = new Intl.NumberFormat(undefined, {
  notation: 'compact',
  maximumFractionDigits: 1,
})

export function formatCount(value: number): string {
  return value >= 10_000 ? compactNumber.format(value) : value.toLocaleString()
}

export function formatDate(value: string | Date): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return date.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })
}

export function formatDateShort(value: string | Date): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return date.toLocaleDateString(undefined, { day: '2-digit', month: 'short' })
}

export function formatTime(value: string | Date): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

export function formatWeekday(value: string | Date): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return date.toLocaleDateString(undefined, { weekday: 'short' })
}

/** Go's zero time, which the api uses to mean "no data yet". */
export function isZeroTime(value: string | undefined | null): boolean {
  if (!value) return true
  return new Date(value).getUTCFullYear() <= 1
}
