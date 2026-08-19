import { useQuery } from '@tanstack/react-query'

import { appUrl } from '@/api/client'
import { EmptyState, Skeleton } from '@/components/ui/State'

type ChartTarget = { userId: string; project?: string }

function chartUrl({ userId, project }: ChartTarget, styled: boolean): string {
  const params = new URLSearchParams()
  if (project) params.set('project', project)
  if (styled) {
    params.set('dark', '')
    params.set('noattr', '')
  }
  const query = params.toString()
  return appUrl(`/api/activity/chart/${encodeURIComponent(userId)}.svg${query ? `?${query}` : ''}`)
}

/**
 * Link to the raw chart, for embedding it somewhere else.
 *
 * Points at the unstyled url on purpose: the dark and noattr flags are this page's
 * presentation choices, not something to hand to a readme.
 */
export function ActivityShareLink(target: ChartTarget) {
  return (
    <a
      href={chartUrl(target, false)}
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
 * The server renders it as an SVG (services/activity.go), so there is nothing to plot
 * here. It is fetched and inlined rather than loaded through an <img>, because an
 * image is an opaque box to the page: the per-day `<title>` tooltips inside the svg
 * only work once the markup is part of the document.
 *
 * Inlining server markup is only safe because this endpoint is ours end to end - every
 * dynamic value in it goes through xml escaping, and its css is scoped to the chart's
 * own group so it cannot repaint the rest of the page.
 */
export function ActivityChart({ userId, project }: ChartTarget) {
  const url = chartUrl({ userId, project }, true)

  const { data, isPending, isError } = useQuery({
    queryKey: ['activity-chart', userId, project ?? ''],
    queryFn: async ({ signal }) => {
      const response = await fetch(url, { credentials: 'same-origin', signal })
      if (!response.ok) throw new Error(`activity chart failed with status ${response.status}`)
      return response.text()
    },
    staleTime: 5 * 60 * 1000,
  })

  if (isPending) return <Skeleton className="h-32 w-full" />
  if (isError || !data) {
    return <EmptyState message="The activity chart could not be loaded." />
  }

  return (
    <div className="w-full overflow-x-auto">
      <div
        className="block min-w-[46rem] [&>svg]:h-auto [&>svg]:max-w-full"
        role="img"
        aria-label={
          project
            ? `Coding activity on ${project} over the past twelve months`
            : 'Coding activity over the past twelve months'
        }
        dangerouslySetInnerHTML={{ __html: data }}
      />
    </div>
  )
}
