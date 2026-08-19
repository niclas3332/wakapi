import { useSearchParams } from 'react-router'

import { appUrl } from '@/api/client'
import { formatDuration } from '@/lib/format'
import { currentInterval } from '@/lib/interval'

type ProjectHeaderProps = {
  project: string
  userId: string
  seconds: number
}

/**
 * Heading shown when the dashboard is filtered down to a single project, carrying
 * the readme badge for that project - the same pairing the legacy page had, and the
 * only place the badge url is offered outside the settings screen.
 */
export function ProjectHeader({ project, userId, seconds }: ProjectHeaderProps) {
  const [search] = useSearchParams()
  const interval = currentInterval(search)

  // "-" is how wakapi spells "heartbeats that carried no project".
  const title = project === '-' ? 'Unknown project' : project
  const badgeUrl = interval
    ? appUrl(
        `/api/badge/${encodeURIComponent(userId)}/interval:${encodeURIComponent(interval)}/project:${encodeURIComponent(project)}`,
      )
    : null

  return (
    <div className="flex flex-wrap items-baseline gap-x-4 gap-y-2">
      <h2 className="text-xl font-semibold text-fg">{title}</h2>
      <span className="tabular text-sm text-fg-muted">{formatDuration(seconds)}</span>
      {badgeUrl && (
        <a
          href={badgeUrl}
          target="_blank"
          rel="noreferrer noopener"
          className="ml-auto"
          title="Badge url for this project"
        >
          <img src={badgeUrl} alt={`Coding time badge for ${title}`} className="h-5" />
        </a>
      )}
    </div>
  )
}
