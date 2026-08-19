import { useMemo, useState } from 'react'

import { colorFor } from '@/lib/colors'
import { formatDateShort, formatDuration, formatDurationCompact } from '@/lib/format'
import type { TimelineItem } from '@/api/types'

const PLOT_HEIGHT_REM = 11

/**
 * Pick a round tick step so the axis reads in whole or half hours rather than in
 * whatever the maximum happens to be.
 */
function axisTicks(maxSeconds: number): { seconds: number; label: string }[] {
  const HOUR = 3600
  const steps = [0.25, 0.5, 1, 2, 3, 4, 6, 8, 12, 24, 48]
  const target = 4
  const step = (steps.find((s) => maxSeconds / (s * HOUR) <= target) ?? steps[steps.length - 1]) * HOUR

  const ticks: { seconds: number; label: string }[] = []
  for (let value = 0; value <= maxSeconds + 1; value += step) {
    // The baseline reads as a plain zero; a unit on it is noise.
    ticks.push({ seconds: value, label: value === 0 ? '0' : formatDurationCompact(value) })
  }
  return ticks
}

/**
 * One stacked column per day, segmented by project.
 *
 * Built from divs rather than a canvas so every segment is a real focusable element
 * with a label - which matters more here than the extra polish a chart library adds.
 */
export function DailyBars({ items }: { items: TimelineItem[] }) {
  const [hovered, setHovered] = useState<{ date: string; project: string; seconds: number } | null>(
    null,
  )

  const { days, scaleMax, ticks, projectColor } = useMemo(() => {
    const totals = new Map<string, number>()
    for (const day of items) {
      for (const project of day.projects) {
        totals.set(project.name, (totals.get(project.name) ?? 0) + project.seconds)
      }
    }
    const ranked = [...totals.entries()].sort((a, b) => b[1] - a[1]).map(([name]) => name)
    const colors = new Map(ranked.map((name, index) => [name, colorFor(name, index)]))

    const days = items.map((day) => ({
      date: day.date,
      total: day.projects.reduce((sum, p) => sum + p.seconds, 0),
      projects: [...day.projects].sort((a, b) => b.seconds - a.seconds),
    }))

    const max = Math.max(...days.map((d) => d.total), 1)
    const ticks = axisTicks(max)
    // Extend the scale to the top tick so the tallest column meets a gridline.
    const scaleMax = Math.max(max, ticks[ticks.length - 1]?.seconds ?? max)

    return { days, scaleMax, ticks, projectColor: colors }
  }, [items])

  return (
    <div>
      <div className="tabular mb-3 h-5 text-xs" aria-live="polite" role="status">
        {hovered ? (
          <>
            <span className="text-fg">{hovered.project}</span>
            <span className="px-1.5 text-fg-subtle">·</span>
            <span className="text-fg-muted">{formatDateShort(hovered.date)}</span>
            <span className="px-1.5 text-fg-subtle">·</span>
            <span className="text-fg-muted">{formatDuration(hovered.seconds)}</span>
          </>
        ) : (
          <span className="text-fg-subtle">Hover a segment for details</span>
        )}
      </div>

      <div className="flex gap-2">
        {/* Value axis: labelled ticks, with gridlines carried across the plot. */}
        <div
          className="tabular relative w-12 shrink-0 text-right text-[10px] text-fg-subtle"
          style={{ height: `${PLOT_HEIGHT_REM}rem` }}
          aria-hidden="true"
        >
          {ticks.map((tick) => (
            <span
              key={tick.seconds}
              className="absolute right-0 -translate-y-1/2 pr-1"
              style={{ bottom: `${(tick.seconds / scaleMax) * 100}%` }}
            >
              {tick.label}
            </span>
          ))}
        </div>

        <div className="relative min-w-0 flex-1" style={{ height: `${PLOT_HEIGHT_REM}rem` }}>
          {ticks.map((tick) => (
            <span
              key={tick.seconds}
              aria-hidden="true"
              className="absolute inset-x-0 border-t border-ink-800"
              style={{ bottom: `${(tick.seconds / scaleMax) * 100}%` }}
            />
          ))}

          <div
            className="absolute inset-0 flex items-end justify-center gap-2 overflow-x-auto"
            onMouseLeave={() => setHovered(null)}
          >
            {days.map((day) => (
              // 2px between stacked segments so adjacent projects stay separable, and
              // a width cap so a short range does not turn into slabs of colour.
              <div
                key={day.date}
                className="flex h-full min-w-6 max-w-20 flex-1 flex-col justify-end gap-[2px]"
              >
                {day.projects.map((project) => {
                  const height = (project.seconds / scaleMax) * 100
                  if (height <= 0) return null
                  return (
                    <button
                      key={project.name}
                      type="button"
                      className="w-full rounded-[2px] transition-[filter] hover:brightness-125 focus-visible:brightness-125"
                      style={{
                        height: `${height}%`,
                        minHeight: '2px',
                        backgroundColor: projectColor.get(project.name),
                      }}
                      onMouseEnter={() =>
                        setHovered({ date: day.date, project: project.name, seconds: project.seconds })
                      }
                      onFocus={() =>
                        setHovered({ date: day.date, project: project.name, seconds: project.seconds })
                      }
                      aria-label={`${project.name}, ${formatDateShort(day.date)}, ${formatDuration(project.seconds)}`}
                    />
                  )
                })}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="tabular mt-1 flex gap-2 pl-14 text-[10px] text-fg-subtle">
        <div className="flex flex-1 justify-center gap-2">
          {days.map((day, index) => (
            <div key={day.date} className="min-w-6 max-w-20 flex-1 text-center">
              {/* Label every other day once the range gets long enough to crowd. */}
              {days.length <= 10 || index % 2 === 0 ? formatDateShort(day.date) : ''}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
