import { useMemo, useState } from 'react'

import { colorFor } from '@/lib/colors'
import { fileName } from '@/lib/entities'
import { formatDuration, formatTime } from '@/lib/format'
import type { HourlyProject } from '@/api/types'
import { EmptyState } from '@/components/ui/State'

const HOURS = 24
const WINDOW_SECONDS = HOURS * 60 * 60
const WINDOW_MS = WINDOW_SECONDS * 1000

type DayStripProps = {
  from: string
  projects: HourlyProject[]
  truncated: boolean
}

type Hovered = {
  project: string
  entity: string
  start: Date
  seconds: number
}

/**
 * A 24-hour track per project, showing when work actually happened.
 *
 * Every other card on the dashboard answers "how much"; this one answers "when",
 * which is the question the raw heartbeat stream is uniquely able to answer. Hovering
 * a segment writes its details into the readout above the tracks rather than a
 * floating tooltip - steadier to read when segments are only a few pixels wide.
 */
export function DayStrip({ from, projects, truncated }: DayStripProps) {
  const [hovered, setHovered] = useState<Hovered | null>(null)

  const start = useMemo(() => new Date(from), [from])
  const ticks = useMemo(
    () =>
      Array.from({ length: 5 }, (_, i) => {
        const at = new Date(start.getTime() + (i * WINDOW_MS) / 4)
        return { at, percent: (i / 4) * 100 }
      }),
    [start],
  )

  if (truncated) {
    return (
      <EmptyState message="This range holds too many separate sessions to draw legibly. Pick a shorter range to see the breakdown." />
    )
  }
  if (projects.length === 0) {
    return <EmptyState message="No activity recorded in the last 24 hours." />
  }

  const ordered = [...projects].sort(
    (a, b) =>
      b.items.reduce((sum, i) => sum + i.seconds, 0) - a.items.reduce((sum, i) => sum + i.seconds, 0),
  )

  return (
    <div>
      <div
        className="tabular mb-3 flex h-5 items-center gap-2 text-xs"
        aria-live="polite"
        role="status"
      >
        {hovered ? (
          <>
            {/* File name only - the full path is a tooltip, not a headline. */}
            <span className="truncate text-fg" title={hovered.entity || undefined}>
              {hovered.entity ? fileName(hovered.entity) : hovered.project}
            </span>
            <span className="text-fg-subtle">·</span>
            <span className="text-fg-muted">{formatTime(hovered.start)}</span>
            <span className="text-fg-subtle">·</span>
            <span className="text-fg-muted">{formatDuration(hovered.seconds)}</span>
          </>
        ) : (
          <span className="text-fg-subtle">Hover a segment for details</span>
        )}
      </div>

      <div className="space-y-2" onMouseLeave={() => setHovered(null)}>
        {ordered.map((project, index) => {
          const color = colorFor(project.project, index)
          const total = project.items.reduce((sum, item) => sum + item.seconds, 0)

          return (
            <div key={project.project} className="grid grid-cols-[7rem_1fr] items-center gap-3 sm:grid-cols-[10rem_1fr]">
              <div className="min-w-0">
                <div className="truncate text-xs text-fg">{project.project}</div>
                <div className="tabular text-[11px] text-fg-subtle">{formatDuration(total)}</div>
              </div>
              <div className="relative h-6 overflow-hidden rounded-sm bg-ink-900">
                {ticks.slice(1, 4).map((tick) => (
                  <span
                    key={tick.percent}
                    aria-hidden="true"
                    className="absolute inset-y-0 w-px bg-ink-700"
                    style={{ left: `${tick.percent}%` }}
                  />
                ))}
                {project.items.map((item, itemIndex) => {
                  const at = new Date(item.from_time)
                  const offset = ((at.getTime() - start.getTime()) / WINDOW_MS) * 100
                  const width = (item.seconds / WINDOW_SECONDS) * 100
                  if (offset < -1 || offset > 101) return null
                  return (
                    <button
                      key={`${item.from_time}-${itemIndex}`}
                      type="button"
                      className="absolute inset-y-1 rounded-[2px] ring-ink-850 transition-[filter] hover:brightness-125 focus-visible:brightness-125"
                      style={{
                        left: `${Math.max(offset, 0)}%`,
                        width: `${Math.max(width, 0.35)}%`,
                        backgroundColor: color,
                      }}
                      onMouseEnter={() =>
                        setHovered({
                          project: project.project,
                          entity: item.entity,
                          start: at,
                          seconds: item.seconds,
                        })
                      }
                      onFocus={() =>
                        setHovered({
                          project: project.project,
                          entity: item.entity,
                          start: at,
                          seconds: item.seconds,
                        })
                      }
                      title={item.entity || undefined}
                      aria-label={`${project.project}, ${item.entity ? fileName(item.entity) : 'unknown file'}, ${formatTime(at)}, ${formatDuration(item.seconds)}`}
                    />
                  )
                })}
              </div>
            </div>
          )
        })}
      </div>

      <div className="tabular relative mt-2 ml-[7.75rem] h-4 text-[11px] text-fg-subtle sm:ml-[11.75rem]">
        {ticks.map((tick, index) => (
          <span
            key={tick.percent}
            className="absolute -translate-x-1/2"
            style={{
              left: `${tick.percent}%`,
              transform: index === 0 ? 'none' : index === ticks.length - 1 ? 'translateX(-100%)' : 'translateX(-50%)',
            }}
          >
            {formatTime(tick.at)}
          </span>
        ))}
      </div>
    </div>
  )
}
