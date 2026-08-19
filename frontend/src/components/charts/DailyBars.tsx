import { useMemo, useState } from 'react'

import { colorFor } from '@/lib/colors'
import { formatDateShort, formatDuration } from '@/lib/format'
import type { TimelineItem } from '@/api/types'

/**
 * One stacked column per day, segmented by project.
 *
 * Built from divs rather than a canvas so every segment is a real focusable element
 * with a label - which matters more here than the extra polish a chart library adds.
 */
export function DailyBars({ items }: { items: TimelineItem[] }) {
  const [hovered, setHovered] = useState<{ date: string; project: string; duration: number } | null>(null)

  const { days, max, projectColor } = useMemo(() => {
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

    return { days, max: Math.max(...days.map((d) => d.total), 1), projectColor: colors }
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
            <span className="text-fg-muted">{formatDuration(hovered.duration)}</span>
          </>
        ) : (
          <span className="text-fg-subtle">Hover a segment for details</span>
        )}
      </div>

      <div
        className="flex h-40 items-end justify-center gap-2 overflow-x-auto pb-1"
        onMouseLeave={() => setHovered(null)}
      >
        {days.map((day) => (
          // 2px between stacked segments so adjacent projects stay separable, and a
          // width cap so a short range does not turn into slabs of colour.
          <div
            key={day.date}
            className="flex h-full min-w-6 max-w-20 flex-1 flex-col justify-end gap-[2px]"
          >
            {day.projects.map((project) => {
              const height = (project.seconds / max) * 100
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
                    setHovered({ date: day.date, project: project.name, duration: project.seconds })
                  }
                  onFocus={() =>
                    setHovered({ date: day.date, project: project.name, duration: project.seconds })
                  }
                  aria-label={`${project.name}, ${formatDateShort(day.date)}, ${formatDuration(project.seconds)}`}
                />
              )
            })}
          </div>
        ))}
      </div>

      <div className="tabular mt-1 flex justify-center gap-2 overflow-x-auto text-[10px] text-fg-subtle">
        {days.map((day, index) => (
          <div key={day.date} className="min-w-6 max-w-20 flex-1 text-center">
            {/* Label every other day once the range gets long enough to crowd. */}
            {days.length <= 10 || index % 2 === 0 ? formatDateShort(day.date) : ''}
          </div>
        ))}
      </div>
    </div>
  )
}
