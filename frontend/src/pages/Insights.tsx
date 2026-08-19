import { useMemo } from 'react'
import { useSearchParams } from 'react-router'

import { useInsights } from '@/api/queries'
import { toApiParams } from '@/lib/interval'
import { formatDate, formatDuration, formatDurationCompact, formatPercent } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/State'
import { RangePicker } from '@/components/ui/RangePicker'
import { StatTile } from '@/components/ui/StatTile'
import { RankedBars } from '@/components/charts/RankedBars'
import { Heatmap } from '@/components/charts/Heatmap'
import { WeekdayBars } from '@/components/charts/WeekdayBars'
import { SplitBar, aiHumanParts } from '@/components/charts/SplitBar'

export function InsightsPage() {
  const [search] = useSearchParams()
  const params = useMemo(() => toApiParams(search), [search])
  const query = useInsights(params)

  if (query.isError) {
    return (
      <ErrorState
        message={`Could not load your insights: ${(query.error as Error).message}`}
        onRetry={() => query.refetch()}
      />
    )
  }

  const data = query.data

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">Insights</h1>
        <div className="ml-auto">
          <RangePicker />
        </div>
      </div>

      {query.isPending || !data ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-[4.75rem]" />
          ))}
        </div>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <StatTile
              label="Total time"
              value={formatDurationCompact(data.total_seconds)}
              note={`across ${data.day_count} ${data.day_count === 1 ? 'day' : 'days'}`}
              emphasis
            />
            <StatTile
              label="Daily average"
              value={formatDurationCompact(data.daily_average)}
              note="every day in range"
            />
            <StatTile
              label="Days coded"
              value={`${data.active_days}`}
              note={
                data.day_count > 0
                  ? `${formatPercent(data.active_days / data.day_count)} of the range`
                  : undefined
              }
            />
            <StatTile
              label="Current streak"
              value={`${data.streak.current}`}
              note={data.streak.current === 1 ? 'day' : 'days'}
            />
            <StatTile
              label="Longest streak"
              value={`${data.streak.longest}`}
              note={
                data.streak.longest_start && data.streak.longest_end
                  ? `${formatDate(data.streak.longest_start)} – ${formatDate(data.streak.longest_end)}`
                  : data.streak.longest === 1
                    ? 'day'
                    : 'days'
              }
              small={Boolean(data.streak.longest_start)}
            />
          </div>

          {data.total_seconds === 0 ? (
            <Card>
              <EmptyState message="Nothing tracked in this range. Pick a longer one to see your patterns." />
            </Card>
          ) : (
            <>
              <Card
                title="Every day"
                meta={
                  data.most_active_day
                    ? `busiest: ${formatDate(data.most_active_day.date)}, ${formatDuration(data.most_active_day.seconds)}`
                    : undefined
                }
              >
                <Heatmap days={data.daily} />
              </Card>

              <div className="grid items-start gap-4 lg:grid-cols-2">
                <Card title="Average weekday">
                  <WeekdayBars weekdays={data.weekdays} />
                </Card>

                <Card title="AI and human" meta="time coded">
                  {/*
                    The split is between the two categories the plugins report. Time
                    spent browsing or writing docs belongs to neither and is left out
                    rather than folded into one of them.
                  */}
                  <SplitBar
                    parts={aiHumanParts(data.ai_seconds, data.human_seconds, formatDuration)}
                  />
                </Card>
              </div>

              <div className="grid items-start gap-4 lg:grid-cols-2 2xl:grid-cols-3">
                <Ranking title="Projects" items={data.projects} />
                <Ranking title="Languages" items={data.languages} colors={data.language_colors} />
                <Ranking title="Editors" items={data.editors} colors={data.editor_colors} />
                <Ranking
                  title="Operating systems"
                  items={data.operating_systems}
                  colors={data.os_colors}
                />
                <Ranking title="Machines" items={data.machines} />
                <Ranking title="Categories" items={data.categories} />
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}

function Ranking({
  title,
  items,
  colors,
}: {
  title: string
  items: { key: string; seconds: number }[]
  colors?: Record<string, string>
}) {
  return (
    <Card title={title} meta={items.length > 0 ? `top ${items.length}` : undefined}>
      <RankedBars items={items} identityColors={colors} initialLimit={10} />
    </Card>
  )
}
