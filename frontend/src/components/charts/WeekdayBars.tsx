import { formatDuration, formatDurationCompact } from '@/lib/format'
import type { WeekdayStat } from '@/api/types'

const NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

/**
 * Average time per weekday.
 *
 * The average divides by every occurrence of that weekday in range, not only the ones
 * that were worked - otherwise "average Sunday" would silently mean "average worked
 * Sunday", a different and much larger number.
 */
export function WeekdayBars({ weekdays }: { weekdays: WeekdayStat[] }) {
  const max = weekdays.reduce((widest, day) => Math.max(widest, day.average_seconds), 0)
  if (max <= 0) {
    return <p className="text-sm text-fg-muted">Nothing recorded in this range.</p>
  }

  // Monday first: a week that starts on Sunday reads oddly next to a work pattern.
  const ordered = [...weekdays].sort((a, b) => ((a.weekday + 6) % 7) - ((b.weekday + 6) % 7))
  const busiest = ordered.reduce((best, day) =>
    day.average_seconds > best.average_seconds ? day : best,
  )

  return (
    <ul className="space-y-2">
      {ordered.map((day) => (
        <li key={day.weekday} className="grid grid-cols-[4.5rem_1fr_4rem] items-center gap-3">
          <span className="text-xs text-fg-muted">{NAMES[day.weekday].slice(0, 3)}</span>
          <span className="h-2 w-full overflow-hidden rounded-full bg-ink-800">
            <span
              className="block h-full rounded-full"
              style={{
                width: `${Math.max((day.average_seconds / max) * 100, day.average_seconds > 0 ? 1.5 : 0)}%`,
                backgroundColor:
                  day.weekday === busiest.weekday ? 'var(--color-accent)' : 'var(--color-series-1)',
              }}
            />
          </span>
          <span
            className="tabular text-right text-xs text-fg-muted"
            title={`${formatDuration(day.average_seconds)} on an average ${NAMES[day.weekday]}, over ${day.days} of them`}
          >
            {formatDurationCompact(day.average_seconds)}
          </span>
        </li>
      ))}
    </ul>
  )
}
