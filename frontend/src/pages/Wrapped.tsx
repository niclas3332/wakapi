import { useEffect } from 'react'
import { useSearchParams } from 'react-router'
import clsx from 'clsx'

import { useWrapped, useWrappedYears } from '@/api/queries'
import { formatDate, formatDuration, formatDurationCompact, formatPercent } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/State'
import { StatTile } from '@/components/ui/StatTile'
import { RankedBars } from '@/components/charts/RankedBars'
import { Heatmap } from '@/components/charts/Heatmap'
import { WeekdayBars } from '@/components/charts/WeekdayBars'
import { SplitBar, aiHumanParts } from '@/components/charts/SplitBar'

export function WrappedPage() {
  const [search, setSearch] = useSearchParams()
  const yearsQuery = useWrappedYears()

  const years = yearsQuery.data ?? []
  const requested = Number(search.get('year'))
  const year = years.includes(requested) ? requested : years[0]

  // Put the year in the url once it is known, so a link to this page is a link to a
  // specific year rather than to "whatever is newest".
  useEffect(() => {
    if (year && !years.includes(requested)) {
      const next = new URLSearchParams(search)
      next.set('year', String(year))
      setSearch(next, { replace: true })
    }
  }, [year, years, requested, search, setSearch])

  const query = useWrapped(year)

  if (yearsQuery.isError) {
    return (
      <ErrorState
        message={`Could not load your years: ${(yearsQuery.error as Error).message}`}
        onRetry={() => yearsQuery.refetch()}
      />
    )
  }

  const data = query.data

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">Wrapped</h1>
        {years.length > 0 && (
          <div className="ml-auto flex flex-wrap gap-1" role="group" aria-label="Year">
            {years.map((option) => (
              <button
                key={option}
                type="button"
                aria-pressed={option === year}
                onClick={() => {
                  const next = new URLSearchParams(search)
                  next.set('year', String(option))
                  setSearch(next)
                }}
                className={clsx(
                  'tabular rounded border px-3 py-1.5 text-sm transition-colors',
                  option === year
                    ? 'border-accent bg-accent-soft text-fg'
                    : 'border-ink-700 text-fg-muted hover:border-ink-600 hover:text-fg',
                )}
              >
                {option}
              </button>
            ))}
          </div>
        )}
      </div>

      {!yearsQuery.isPending && years.length === 0 && (
        <Card>
          <EmptyState message="Nothing tracked yet. Your first year appears here once an editor sends its first heartbeat." />
        </Card>
      )}

      {query.isError && (
        <ErrorState
          message={`Could not load ${year}: ${(query.error as Error).message}`}
          onRetry={() => query.refetch()}
        />
      )}

      {(query.isPending || !data) && years.length > 0 && !query.isError && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-[4.75rem]" />
          ))}
        </div>
      )}

      {data && (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <StatTile
              label={`${data.year} in total`}
              value={formatDurationCompact(data.total_seconds)}
              note={`${data.active_days} of ${data.day_count} days`}
              emphasis
            />
            <StatTile
              label="Daily average"
              value={formatDurationCompact(data.daily_average)}
              note="every day of the year"
            />
            <StatTile
              label="Longest streak"
              value={`${data.streak.longest}`}
              note={
                data.streak.longest_start && data.streak.longest_end
                  ? `${formatDate(data.streak.longest_start)} – ${formatDate(data.streak.longest_end)}`
                  : 'days'
              }
              small={Boolean(data.streak.longest_start)}
            />
            <StatTile
              label="Busiest day"
              value={
                data.most_active_day ? formatDurationCompact(data.most_active_day.seconds) : '—'
              }
              note={data.most_active_day ? formatDate(data.most_active_day.date) : undefined}
            />
          </div>

          {data.total_seconds === 0 ? (
            <Card>
              <EmptyState message={`Nothing tracked in ${data.year}.`} />
            </Card>
          ) : (
            <>
              <Card title="The year" meta={`${data.active_days} days coded`}>
                <Heatmap days={data.daily} />
              </Card>

              <div className="grid items-start gap-4 lg:grid-cols-2">
                <Card title="Average weekday">
                  <WeekdayBars weekdays={data.weekdays} />
                </Card>
                <Card
                  title="AI and human"
                  meta={
                    data.ai_seconds + data.human_seconds > 0
                      ? `${formatPercent(data.ai_seconds / (data.ai_seconds + data.human_seconds))} AI`
                      : undefined
                  }
                >
                  <SplitBar
                    parts={aiHumanParts(data.ai_seconds, data.human_seconds, formatDuration)}
                  />
                </Card>
              </div>

              <div className="grid items-start gap-4 lg:grid-cols-2 2xl:grid-cols-4">
                <Card title="Projects">
                  <RankedBars items={data.projects} initialLimit={10} />
                </Card>
                <Card title="Languages">
                  <RankedBars
                    items={data.languages}
                    identityColors={data.language_colors}
                    initialLimit={10}
                  />
                </Card>
                <Card title="Editors">
                  <RankedBars
                    items={data.editors}
                    identityColors={data.editor_colors}
                    initialLimit={10}
                  />
                </Card>
                <Card title="Operating systems">
                  <RankedBars
                    items={data.operating_systems}
                    identityColors={data.os_colors}
                    initialLimit={10}
                  />
                </Card>
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}
