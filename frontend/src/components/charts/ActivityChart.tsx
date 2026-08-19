import { useState } from 'react'

import { appUrl } from '@/api/client'
import { Skeleton } from '@/components/ui/State'

function chartUrl(userId: string, styled: boolean): string {
  const path = `/api/activity/chart/${encodeURIComponent(userId)}.svg`
  return appUrl(styled ? `${path}?dark&noattr` : path)
}

/**
 * Link to the raw chart, for embedding it somewhere else.
 *
 * Points at the unstyled url on purpose: the dark and noattr flags are this page's
 * presentation choices, not something to hand to a readme.
 */
export function ActivityShareLink({ userId }: { userId: string }) {
  return (
    <a
      href={chartUrl(userId, false)}
      target="_blank"
      rel="noreferrer noopener"
      title="Open the chart image, e.g. to embed it in a readme"
      className="rounded border border-ink-700 px-2 py-1 text-xs text-fg-muted transition-colors hover:border-ink-600 hover:text-fg"
    >
      Share
    </a>
  )
}

/**
 * The GitHub-style contribution grid for the past twelve months.
 *
 * The server renders it as an SVG (services/activity.go), so there is nothing to
 * plot here. It is loaded as an image rather than injected as markup, so no
 * server-produced html reaches innerHTML.
 *
 * The range picker does not apply - this endpoint only knows one window, which is
 * why the card says so.
 */
export function ActivityChart({ userId }: { userId: string }) {
  const [state, setState] = useState<'loading' | 'ready' | 'failed'>('loading')

  return (
    <div className="w-full overflow-x-auto">
      {state === 'loading' && <Skeleton className="h-32 w-full" />}
      {state === 'failed' && (
        <p className="py-8 text-center text-sm text-fg-muted">
          The activity chart could not be loaded.
        </p>
      )}
      <img
        src={chartUrl(userId, true)}
        alt="Coding activity over the past twelve months"
        className={state === 'ready' ? 'block min-w-[46rem] max-w-full' : 'hidden'}
        onLoad={() => setState('ready')}
        onError={() => setState('failed')}
      />
    </div>
  )
}
