import { useState } from 'react'

import { appUrl } from '@/api/client'
import { Skeleton } from '@/components/ui/State'

/**
 * The GitHub-style contribution grid for the past twelve months.
 *
 * The server renders it as an SVG (services/activity.go), so there is nothing to
 * plot here. It is loaded as an image rather than injected as markup: the browser
 * gets caching and lazy loading for free, and no server-produced markup ever
 * touches innerHTML.
 *
 * The range picker does not apply - this endpoint only knows one window, which is
 * why the card says so.
 */
export function ActivityChart({ userId }: { userId: string }) {
  const [state, setState] = useState<'loading' | 'ready' | 'failed'>('loading')

  const src = appUrl(`/api/activity/chart/${encodeURIComponent(userId)}.svg?dark&noattr`)

  return (
    <div className="w-full overflow-x-auto">
      {state === 'loading' && <Skeleton className="h-32 w-full" />}
      {state === 'failed' && (
        <p className="py-8 text-center text-sm text-fg-muted">
          The activity chart could not be loaded.
        </p>
      )}
      <img
        src={src}
        alt="Coding activity over the past twelve months"
        className={state === 'ready' ? 'block min-w-[46rem] max-w-full' : 'hidden'}
        onLoad={() => setState('ready')}
        onError={() => setState('failed')}
      />
    </div>
  )
}
