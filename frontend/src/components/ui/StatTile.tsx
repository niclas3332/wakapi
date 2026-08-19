import type { ReactNode } from 'react'
import clsx from 'clsx'

type StatTileProps = {
  label: string
  value: ReactNode
  /** One line of context under the value, e.g. what the number is averaged over. */
  note?: ReactNode
  /** Draws attention to the primary figure of the screen. */
  emphasis?: boolean
  /** For values that are names rather than numbers, which need room to breathe. */
  small?: boolean
}

export function StatTile({ label, value, note, emphasis = false, small = false }: StatTileProps) {
  return (
    <div
      className={clsx(
        'min-w-0 rounded-lg border px-4 py-3',
        emphasis ? 'border-accent/30 bg-accent-soft' : 'border-ink-700 bg-ink-850',
      )}
    >
      <div className="eyebrow">{label}</div>
      <div
        className={clsx(
          'tabular mt-1 truncate font-medium text-fg',
          emphasis ? 'text-2xl' : small ? 'text-base' : 'text-xl',
        )}
        title={typeof value === 'string' ? value : undefined}
      >
        {value}
      </div>
      {note && <div className="mt-0.5 truncate text-xs text-fg-subtle">{note}</div>}
    </div>
  )
}
