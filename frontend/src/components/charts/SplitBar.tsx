import { AI_COLOR, HUMAN_COLOR } from '@/lib/colors'
import { formatPercent } from '@/lib/format'

type Part = {
  label: string
  value: number
  color: string
  /** Formatted value shown beside the label. */
  display: string
}

/**
 * Two quantities as one bar plus a legend.
 *
 * A bar rather than a donut: the question is "what share", and a straight length is
 * read more accurately than an angle. Every part is labelled with its own name and
 * value, so identity never rests on colour.
 */
export function SplitBar({ parts }: { parts: Part[] }) {
  const total = parts.reduce((sum, part) => sum + part.value, 0)

  if (total <= 0) {
    return <p className="text-sm text-fg-muted">Nothing recorded in this range.</p>
  }

  return (
    <div>
      {/* 2px gaps in the surface colour keep adjacent segments separable. */}
      <div className="flex h-2.5 w-full gap-[2px] overflow-hidden rounded-full">
        {parts.map((part) =>
          part.value > 0 ? (
            <span
              key={part.label}
              className="h-full first:rounded-l-full last:rounded-r-full"
              style={{ width: `${(part.value / total) * 100}%`, backgroundColor: part.color }}
            />
          ) : null,
        )}
      </div>

      <ul className="mt-3 space-y-1.5">
        {parts.map((part) => (
          <li key={part.label} className="flex items-baseline gap-2 text-sm">
            <span
              aria-hidden="true"
              className="inline-block h-2.5 w-2.5 shrink-0 rounded-sm"
              style={{ backgroundColor: part.color }}
            />
            <span className="text-fg">{part.label}</span>
            <span className="tabular ml-auto text-fg-muted">{part.display}</span>
            <span className="tabular w-12 shrink-0 text-right text-fg-subtle">
              {formatPercent(part.value / total)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** The AI/human split, whose two colours mean the same thing on every screen. */
export function aiHumanParts(
  ai: number,
  human: number,
  format: (value: number) => string,
): Part[] {
  return [
    { label: 'AI', value: ai, color: AI_COLOR, display: format(ai) },
    { label: 'Human', value: human, color: HUMAN_COLOR, display: format(human) },
  ]
}
