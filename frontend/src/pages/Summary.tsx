import { useMemo } from 'react'
import { useSearchParams } from 'react-router'

import { useHourly, useSummary, useTimeline } from '@/api/queries'
import type { Summary, SummaryItem } from '@/api/types'
import { appUrl } from '@/api/client'
import { AI_COLOR, HUMAN_COLOR } from '@/lib/colors'
import { formatDuration, formatDurationCompact, formatPercent, isZeroTime } from '@/lib/format'
import { toApiParams } from '@/lib/interval'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingRows, Skeleton } from '@/components/ui/State'
import { FilterBar } from '@/components/ui/FilterBar'
import { RangePicker } from '@/components/ui/RangePicker'
import { StatTile } from '@/components/ui/StatTile'
import { RankedBars } from '@/components/charts/RankedBars'
import { DayStrip } from '@/components/charts/DayStrip'
import { DailyBars } from '@/components/charts/DailyBars'

const MS_PER_DAY = 24 * 60 * 60 * 1000

export function SummaryPage() {
  const [search, setSearch] = useSearchParams()
  const params = useMemo(() => toApiParams(search), [search])

  const summaryQuery = useSummary(params)
  const timelineQuery = useTimeline(params)
  const hourlyQuery = useHourly(params)

  const summary = summaryQuery.data?.summary
  const stats = useMemo(() => (summary ? deriveStats(summary) : null), [summary])

  function applyFilter(key: string, value: string) {
    const next = new URLSearchParams(search)
    next.set(key, value)
    setSearch(next)
  }

  if (summaryQuery.isError) {
    return (
      <ErrorState
        message={`Could not load your summary: ${(summaryQuery.error as Error).message}`}
        onRetry={() => summaryQuery.refetch()}
      />
    )
  }

  const isSingleDay = stats !== null && stats.days <= 1
  const noData = summary && stats && stats.total === 0
  // Only judge this once the answer has actually arrived - otherwise the onboarding
  // card flashes on every load, because "no first heartbeat yet" and "not loaded yet"
  // look identical.
  const neverAnyData = summaryQuery.isSuccess && isZeroTime(summaryQuery.data.user_first_data)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">Dashboard</h1>
        <div className="ml-auto">
          <RangePicker />
        </div>
      </div>

      <FilterBar filters={summaryQuery.data?.available_filters} />

      {summaryQuery.isPending || !stats || !summary ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-[4.75rem]" />
          ))}
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          <StatTile
            label="Total time"
            value={formatDurationCompact(stats.total)}
            note={`${stats.days} ${stats.days === 1 ? 'day' : 'days'} in range`}
            emphasis
          />
          <StatTile
            label="Daily average"
            value={formatDurationCompact(stats.dailyAverage)}
            note={`over ${stats.days} ${stats.days === 1 ? 'day' : 'days'}`}
          />
          <StatTile label="Top project" value={stats.topProject ?? '—'} note={stats.topProjectTime} small />
          <StatTile label="Top language" value={stats.topLanguage ?? '—'} note={stats.topLanguageTime} small />
          <StatTile label="Top editor" value={stats.topEditor ?? '—'} note={stats.topEditorTime} small />
          <StatTile
            label="AI coding"
            value={stats.aiRatio === null ? '—' : formatPercent(stats.aiRatio)}
            note={
              stats.aiRatio === null ? (
                'no category data'
              ) : (
                <span className="flex items-center gap-1.5">
                  <span
                    aria-hidden="true"
                    className="inline-block h-2 w-2 rounded-full"
                    style={{ backgroundColor: AI_COLOR }}
                  />
                  AI
                  <span
                    aria-hidden="true"
                    className="ml-1.5 inline-block h-2 w-2 rounded-full"
                    style={{ backgroundColor: HUMAN_COLOR }}
                  />
                  human
                </span>
              )
            }
          />
        </div>
      )}

      {neverAnyData && (
        <Card title="No data yet">
          <EmptyState
            message="Nothing has been tracked for this account. Install a WakaTime plugin in your editor and point it at this instance."
            action={
              <a
                href={appUrl('/setup')}
                className="rounded bg-accent-strong px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent"
              >
                Connect an editor
              </a>
            }
          />
        </Card>
      )}

      {noData && !neverAnyData && (
        <Card>
          <EmptyState message="No activity in this range. Try a longer range or clear your filters." />
        </Card>
      )}

      {!noData && (
        <>
          {timelineQuery.data?.available && timelineQuery.data.items.length > 0 && (
            <Card title="Daily activity" meta="by project">
              <DailyBars items={timelineQuery.data.items} />
            </Card>
          )}

          {/*
            The hourly breakdown always covers at most the last 24 hours, so for a
            longer range it is a detail view rather than the headline - which is why
            it sits below the daily chart and says so in its title.
          */}
          <Card title={isSingleDay ? 'When you coded' : 'Last 24 hours'} meta="by project">
            {hourlyQuery.isPending ? (
              <LoadingRows rows={4} />
            ) : hourlyQuery.isError ? (
              <ErrorState
                message="Could not load the hourly breakdown."
                onRetry={() => hourlyQuery.refetch()}
              />
            ) : (
              <DayStrip
                from={hourlyQuery.data!.from}
                projects={hourlyQuery.data!.items}
                truncated={hourlyQuery.data!.truncated}
              />
            )}
          </Card>

          <div className="grid items-start gap-4 lg:grid-cols-2 2xl:grid-cols-3">
            <Breakdown
              title="Projects"
              items={summary?.projects}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('project', value)}
            />
            <Breakdown
              title="Languages"
              items={summary?.languages}
              colors={summaryQuery.data?.language_colors}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('language', value)}
            />
            <Breakdown
              title="Editors"
              items={summary?.editors}
              colors={summaryQuery.data?.editor_colors}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('editor', value)}
            />
            <Breakdown
              title="Operating systems"
              items={summary?.operating_systems}
              colors={summaryQuery.data?.os_colors}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('operating_system', value)}
            />
            <Breakdown
              title="Machines"
              items={summary?.machines}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('machine', value)}
            />
            <Breakdown
              title="Categories"
              items={summary?.categories}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('category', value)}
            />
            <Breakdown
              title="Labels"
              items={summary?.labels}
              loading={summaryQuery.isPending}
              onSelect={(value) => applyFilter('label', value)}
            />
            {/* Branches and files are only computed when a project filter is active. */}
            {summary && summary.branches.length > 0 && (
              <Breakdown
                title="Branches"
                items={summary.branches}
                loading={false}
                onSelect={(value) => applyFilter('branch', value)}
              />
            )}
            {summary && summary.entities.length > 0 && (
              <Breakdown title="Files" items={summary.entities} loading={false} />
            )}
          </div>
        </>
      )}
    </div>
  )
}

type BreakdownProps = {
  title: string
  items?: SummaryItem[]
  colors?: Record<string, string>
  loading: boolean
  onSelect?: (value: string) => void
}

function Breakdown({ title, items, colors, loading, onSelect }: BreakdownProps) {
  return (
    <Card title={title} meta={items && items.length > 0 ? `${items.length}` : undefined}>
      {loading || !items ? (
        <LoadingRows rows={6} />
      ) : (
        <RankedBars items={items} identityColors={colors} onSelect={onSelect} />
      )}
    </Card>
  )
}

function topOf(items: SummaryItem[]): SummaryItem | undefined {
  return items.length === 0 ? undefined : items.reduce((a, b) => (b.seconds > a.seconds ? b : a))
}

function deriveStats(summary: Summary) {
  const total = summary.projects.reduce((sum, item) => sum + item.seconds, 0)

  const from = new Date(summary.from).getTime()
  const to = new Date(summary.to).getTime()
  // Partial days count as a day, matching how the range reads to a person.
  const days = Math.max(1, Math.ceil((to - from) / MS_PER_DAY))

  const byCategory = new Map(summary.categories.map((item) => [item.key.toLowerCase(), item.seconds]))
  const ai = byCategory.get('ai coding') ?? 0
  const human = byCategory.get('coding') ?? 0
  // Same definition the legacy dashboard used: ai coding against ai coding + coding.
  const aiRatio = ai + human > 0 ? ai / (ai + human) : null

  const topProject = topOf(summary.projects)
  const topLanguage = topOf(summary.languages)
  const topEditor = topOf(summary.editors)

  return {
    total,
    days,
    dailyAverage: total / days,
    aiRatio,
    topProject: topProject?.key,
    topProjectTime: topProject ? formatDuration(topProject.seconds) : undefined,
    topLanguage: topLanguage?.key,
    topLanguageTime: topLanguage ? formatDuration(topLanguage.seconds) : undefined,
    topEditor: topEditor?.key,
    topEditorTime: topEditor ? formatDuration(topEditor.seconds) : undefined,
  }
}
