import { useState } from 'react'

import { colorFor, OTHER_KEY, sliceTop } from '@/lib/colors'
import { formatDuration } from '@/lib/format'
import type { SummaryItem } from '@/api/types'
import { EmptyState } from '@/components/ui/State'

type RankedBarsProps = {
  items: SummaryItem[]
  /** Identity colours from the server, where the entity type has them. */
  identityColors?: Record<string, string>
  /** How many rows to show before the "show all" control. */
  initialLimit?: number
  /** Called when a row is activated, e.g. to apply it as a filter. */
  onSelect?: (key: string) => void
  emptyMessage?: string
}

/**
 * A ranked list with a magnitude bar behind each row.
 *
 * Chosen over a donut for every categorical breakdown: the job is comparing
 * magnitudes and reading names, which a sorted bar does directly. Every row is
 * labelled, so identity never depends on colour alone.
 */
export function RankedBars({
  items,
  identityColors,
  initialLimit = 8,
  onSelect,
  emptyMessage = 'Nothing tracked in this range.',
}: RankedBarsProps) {
  const [expanded, setExpanded] = useState(false)

  if (items.length === 0) return <EmptyState message={emptyMessage} />

  const sorted = [...items].sort((a, b) => b.seconds - a.seconds)
  const grandTotal = sorted.reduce((sum, item) => sum + item.seconds, 0)

  const { shown, otherSeconds, otherCount } = expanded
    ? { shown: sorted, otherSeconds: 0, otherCount: 0 }
    : sliceTop(sorted, initialLimit)

  const rows = [...shown]
  if (otherSeconds > 0) rows.push({ key: OTHER_KEY, seconds: otherSeconds })

  // Scale against the widest row that is actually drawn, not against the largest
  // single entry: "Other" is a sum and routinely outgrows the top entry, which
  // would push its bar past the full width of the track.
  const max = rows.reduce((widest, item) => Math.max(widest, item.seconds), 0)

  return (
    <div>
      <ul className="space-y-1.5">
        {rows.map((item, index) => {
          const share = grandTotal > 0 ? item.seconds / grandTotal : 0
          const width = max > 0 ? Math.min(100, Math.max((item.seconds / max) * 100, 1.5)) : 0
          const color = colorFor(item.key, index, identityColors)
          const isOther = item.key === OTHER_KEY
          const label = isOther ? `Other (${otherCount})` : item.key
          const interactive = Boolean(onSelect) && !isOther

          // Label and value sit on the card surface in text tokens, never on the
          // mark and never in the series colour. The bar is a thin rule underneath,
          // so a long name stays readable no matter how wide its magnitude is.
          const content = (
            <>
              <div className="flex w-full items-baseline justify-between gap-3">
                <span className="truncate text-sm text-fg">{label}</span>
                <span className="tabular shrink-0 text-xs text-fg-muted">
                  {formatDuration(item.seconds)}
                  <span className="ml-2 text-fg-subtle">
                    {(share * 100).toFixed(share < 0.1 ? 1 : 0)}%
                  </span>
                </span>
              </div>
              <div className="mt-1.5 h-[3px] w-full overflow-hidden rounded-full bg-ink-800">
                <div
                  aria-hidden="true"
                  className="h-full rounded-full"
                  style={{ width: `${width}%`, backgroundColor: color }}
                />
              </div>
            </>
          )

          return (
            <li key={item.key}>
              {interactive ? (
                <button
                  type="button"
                  onClick={() => onSelect?.(item.key)}
                  title={`Filter by ${label}`}
                  className="block w-full rounded-sm px-2 py-1.5 text-left transition-colors hover:bg-ink-800"
                >
                  {content}
                </button>
              ) : (
                <div className="rounded-sm px-2 py-1.5">{content}</div>
              )}
            </li>
          )
        })}
      </ul>

      {sorted.length > initialLimit && (
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          className="mt-3 text-xs text-accent transition-colors hover:text-fg"
        >
          {expanded ? 'Show top ' + initialLimit : `Show all ${sorted.length}`}
        </button>
      )}
    </div>
  )
}
