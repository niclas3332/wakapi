import { useMemo, useState } from 'react'

import { formatDate, formatDuration } from '@/lib/format'
import type { DayTotal } from '@/api/types'

const LEVELS = 4
const WEEKDAY_LABELS = ['', 'Mon', '', 'Wed', '', 'Fri', '']

/**
 * Shading levels cut at quartiles of the days that were actually worked.
 *
 * A linear scale against the busiest day is unreadable as soon as one outlier exists:
 * against a 16 hour day, a full workday sits at half brightness. Ranking a day among
 * the user's own days is what makes the grid say something. Same rule the server-side
 * activity chart uses, so the two look alike.
 */
function levelsFor(values: number[]): (value: number) => number {
  const active = values.filter((value) => value > 0).sort((a, b) => a - b)
  if (active.length === 0) return () => 0

  const quantile = (q: number) => {
    if (active.length === 1) return active[0]
    const position = q * (active.length - 1)
    const lower = Math.floor(position)
    const upper = Math.ceil(position)
    if (lower === upper) return active[lower]
    return active[lower] + (position - lower) * (active[upper] - active[lower])
  }

  const cuts = Array.from({ length: LEVELS - 1 }, (_, i) => quantile((i + 1) / LEVELS))
  return (value: number) => {
    if (value <= 0) return 0
    return 1 + cuts.filter((cut) => value > cut).length
  }
}

/**
 * A year grid of days, one column per week.
 *
 * Built from divs rather than an image so every cell is a real element with its own
 * title - the reason the dashboard's server-rendered chart is inlined too.
 */
export function Heatmap({ days }: { days: DayTotal[] }) {
  const [hovered, setHovered] = useState<DayTotal | null>(null)

  const { weeks, level } = useMemo(() => {
    const level = levelsFor(days.map((day) => day.seconds))
    if (days.length === 0) return { weeks: [] as (DayTotal | null)[][], level }

    // Pad the first week so weekdays line up in rows.
    const leading = new Date(days[0].date).getDay()
    const cells: (DayTotal | null)[] = [...Array(leading).fill(null), ...days]

    const weeks: (DayTotal | null)[][] = []
    for (let i = 0; i < cells.length; i += 7) {
      const week = cells.slice(i, i + 7)
      while (week.length < 7) week.push(null)
      weeks.push(week)
    }
    return { weeks, level }
  }, [days])

  if (days.length === 0) return null

  return (
    <div>
      <div className="tabular mb-2 h-5 text-xs" aria-live="polite" role="status">
        {hovered ? (
          <>
            <span className="text-fg">{formatDuration(hovered.seconds)}</span>
            <span className="px-1.5 text-fg-subtle">·</span>
            <span className="text-fg-muted">{formatDate(hovered.date)}</span>
          </>
        ) : (
          <span className="text-fg-subtle">Hover a day for details</span>
        )}
      </div>

      <div className="flex gap-2 overflow-x-auto pb-1">
        <div className="flex shrink-0 flex-col gap-[3px] pt-0.5" aria-hidden="true">
          {WEEKDAY_LABELS.map((label, index) => (
            <span key={index} className="h-3 text-[10px] leading-3 text-fg-subtle">
              {label}
            </span>
          ))}
        </div>

        <div className="flex gap-[3px]" onMouseLeave={() => setHovered(null)}>
          {weeks.map((week, weekIndex) => (
            <div key={weekIndex} className="flex flex-col gap-[3px]">
              {week.map((day, dayIndex) => (
                <span
                  key={dayIndex}
                  className="h-3 w-3 rounded-[2px]"
                  style={{
                    backgroundColor: day
                      ? `color-mix(in oklab, var(--color-accent) ${level(day.seconds) * 25}%, var(--color-ink-800))`
                      : 'transparent',
                  }}
                  title={day ? `${formatDuration(day.seconds)} on ${formatDate(day.date)}` : undefined}
                  onMouseEnter={() => day && setHovered(day)}
                />
              ))}
            </div>
          ))}
        </div>
      </div>

      <div className="mt-2 flex items-center gap-1.5 text-[10px] text-fg-subtle">
        <span>Less</span>
        {Array.from({ length: LEVELS + 1 }, (_, index) => (
          <span
            key={index}
            className="h-3 w-3 rounded-[2px]"
            style={{
              backgroundColor: `color-mix(in oklab, var(--color-accent) ${index * 25}%, var(--color-ink-800))`,
            }}
          />
        ))}
        <span>More</span>
        <span className="ml-2">quartiles of the days you worked</span>
      </div>
    </div>
  )
}
