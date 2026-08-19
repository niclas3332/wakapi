import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'

import { useProjects } from '@/api/queries'
import { stableColorFor } from '@/lib/colors'
import { formatCount, formatDate } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingRows } from '@/components/ui/State'

const PAGE_SIZE = 24

export function ProjectsPage() {
  const [search, setSearch] = useSearchParams()
  const query = search.get('q') ?? ''
  const page = Number(search.get('page') ?? '1') || 1
  const [draft, setDraft] = useState(query)

  const projectsQuery = useProjects(page, PAGE_SIZE, query)

  function submitSearch(value: string) {
    const next = new URLSearchParams(search)
    if (value) next.set('q', value)
    else next.delete('q')
    next.delete('page')
    setSearch(next)
  }

  function goToPage(next: number) {
    const params = new URLSearchParams(search)
    params.set('page', String(next))
    setSearch(params)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">Projects</h1>
        <form
          className="ml-auto flex items-center gap-2"
          onSubmit={(event) => {
            event.preventDefault()
            submitSearch(draft.trim())
          }}
        >
          <input
            type="search"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="Search projects"
            aria-label="Search projects"
            className="w-48 rounded border border-ink-700 bg-ink-850 px-3 py-1.5 text-sm text-fg placeholder:text-fg-subtle sm:w-64"
          />
          <button
            type="submit"
            className="rounded border border-ink-700 px-3 py-1.5 text-sm text-fg-muted transition-colors hover:border-ink-600 hover:text-fg"
          >
            Search
          </button>
        </form>
      </div>

      <Card
        meta={
          query ? (
            <>
              matching <span className="text-fg">{query}</span>
            </>
          ) : undefined
        }
        title="All projects"
      >
        {projectsQuery.isPending ? (
          <LoadingRows rows={8} />
        ) : projectsQuery.isError ? (
          <ErrorState
            message={`Could not load projects: ${(projectsQuery.error as Error).message}`}
            onRetry={() => projectsQuery.refetch()}
          />
        ) : projectsQuery.data!.items.length === 0 ? (
          <EmptyState
            message={
              query
                ? `No project matches "${query}".`
                : 'No projects yet. They appear once your editor sends its first heartbeat.'
            }
          />
        ) : (
          <ul className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
            {projectsQuery.data!.items.map((project) => (
              <li key={project.project}>
                <Link
                  to={`/summary?interval=any&project=${encodeURIComponent(project.project)}`}
                  className="block rounded-lg border border-ink-700 bg-ink-900 p-3 transition-colors hover:border-ink-600 hover:bg-ink-850"
                >
                  <div className="flex items-center gap-2">
                    <span
                      aria-hidden="true"
                      className="h-2.5 w-2.5 shrink-0 rounded-sm"
                      style={{ backgroundColor: stableColorFor(project.project) }}
                    />
                    <span className="truncate text-sm font-medium text-fg">{project.project}</span>
                    <span className="tabular ml-auto shrink-0 text-xs text-fg-subtle">
                      {formatCount(project.count)}
                    </span>
                  </div>
                  <div className="mt-2 flex items-center gap-2 text-xs text-fg-muted">
                    <span className="truncate">{project.top_language || 'unknown'}</span>
                    <span className="tabular ml-auto shrink-0 text-fg-subtle">
                      {formatDate(project.last)}
                    </span>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => goToPage(page - 1)}
          disabled={page <= 1}
          className="rounded border border-ink-700 px-3 py-1.5 text-sm text-fg-muted transition-colors hover:text-fg disabled:opacity-40"
        >
          Previous
        </button>
        <span className="tabular text-xs text-fg-subtle">Page {page}</span>
        <button
          type="button"
          onClick={() => goToPage(page + 1)}
          disabled={!projectsQuery.data?.has_more}
          className="rounded border border-ink-700 px-3 py-1.5 text-sm text-fg-muted transition-colors hover:text-fg disabled:opacity-40"
        >
          Next
        </button>
      </div>
    </div>
  )
}
