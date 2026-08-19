import type { ReactNode } from 'react'
import clsx from 'clsx'

/** Placeholder block that matches the height of the content it stands in for. */
export function Skeleton({ className }: { className?: string }) {
  return <div className={clsx('animate-pulse rounded bg-ink-800', className)} aria-hidden="true" />
}

export function LoadingRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-3" role="status" aria-label="Loading">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className="h-4 w-full" />
      ))}
    </div>
  )
}

/**
 * An empty screen is an invitation to act, so it says what is missing and what to
 * do about it rather than just "no data".
 */
export function EmptyState({ message, action }: { message: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 px-4 py-10 text-center">
      <p className="text-sm text-fg-muted">{message}</p>
      {action}
    </div>
  )
}

/** Errors state what happened and how to recover, in the interface's voice. */
export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 px-4 py-10 text-center">
      <p className="text-sm text-danger">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded border border-ink-600 px-3 py-1.5 text-sm text-fg transition-colors hover:bg-ink-800"
        >
          Try again
        </button>
      )}
    </div>
  )
}
