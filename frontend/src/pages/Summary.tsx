import { useMemo } from 'react'
import { useSearchParams } from 'react-router'

import { useBootstrap, useHourly, useSummary, useTimeline } from '@/api/queries'
import type { Summary, SummaryItem } from '@/api/types'
import { appUrl } from '@/api/client'
import { AI_COLOR, HUMAN_COLOR } from '@/lib/colors'
import {
  formatCount,
  formatDuration,
  formatDurationCompact,
  formatPercent,
  isZeroTime,
} from '@/lib/format'
import { toApiParams } from '@/lib/interval'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingRows, Skeleton } from '@/components/ui/State'
import { FilterBar } from '@/components/ui/FilterBar'
import { RangePicker } from '@/components/ui/RangePicker'
import { StatTile } from '@/components/ui/StatTile'
import { ProjectHeader } from '@/components/ui/ProjectHeader'
import { RankedBars } from '@/components/charts/RankedBars'
import { DayStrip } from '@/components/charts/DayStrip'
import { DailyBars } from '@/components/charts/DailyBars'
import { ActivityChart, ActivityShareLink } from '@/components/charts/ActivityChart'
import { fileName } from '@/lib/entities'

const MS_PER_DAY = 24 * 60 * 60 * 1000

export function SummaryPage() {
  const [search, setSearch] = useSearchParams()
  const params = useMemo(() => toApiParams(search), [search])

  const summaryQuery = useSummary(params)
  const timelineQuery = useTimeline(params)
  const hourlyQuery = useHourly(params)
  const { data: bootstrap } = useBootstrap()

  const projectFilter = search.get('project') ?? ''

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

      {projectFilter && summary && stats && bootstrap && (
        <ProjectHeader project={projectFilter} userId={bootstrap.user.id} seconds={stats.total} />
      )}

      {summaryQuery.isPending || !stats || !summary ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-7">
          {Array.from({ length: 7 }, (_, i) => (
            <Skeleton key={i} className="h-[4.75rem]" />
          ))}
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-7">
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
          <StatTile
            label="Heartbeats"
            value={formatCount(summary.num_heartbeats)}
            note="pings received"
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

      {/* Only worth saying when a retention policy is actually in force. */}
      {summaryQuery.data && summaryQuery.data.data_retention_months > 0 && (
        <p className="rounded-lg border border-ink-700 bg-ink-850 px-4 py-3 text-xs text-fg-muted">
          Coding activity older than {summaryQuery.data.data_retention_months} months is deleted on
          this instance.{' '}
          <a href={appUrl('/settings#subscription')} className="text-accent hover:text-fg">
            Subscription settings
          </a>
        </p>
      )}

      {!noData && (
        <>
          {/*
            The activity grid always covers the past twelve months, independent of
            the range picker, so it sits at the top as the constant frame of
            reference rather than pretending to follow the selected range.
          */}
          {bootstrap && (
            <Card
              title="Activity"
              meta="past 12 months"
              actions={<ActivityShareLink userId={bootstrap.user.id} />}
            >
              <ActivityChart userId={bootstrap.user.id} />
            </Card>
          )}

          <Card title="Timeline" meta="by project, per day">
            {timelineQuery.isPending ? (
              <LoadingRows rows={4} />
            ) : timelineQuery.data?.available && timelineQuery.data.items.length > 0 ? (
              <DailyBars items={timelineQuery.data.items} />
            ) : (
              <EmptyState
                message={`A per-day timeline needs a range between ${timelineQuery.data?.min_range_days ?? 3} and ${timelineQuery.data?.max_range_days ?? 31} days. Pick one to see it.`}
              />
            )}
          </Card>

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
              // Entity keys are absolute paths, so the row shows the file name and
              // keeps the path as a tooltip.
              <Breakdown
                title="Files"
                items={summary.entities}
                loading={false}
                formatLabel={fileName}
              />
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
  formatLabel?: (key: string) => string
}

function Breakdown({ title, items, colors, loading, onSelect, formatLabel }: BreakdownProps) {
  return (
    <Card title={title} meta={items && items.length > 0 ? `${items.length}` : undefined}>
      {loading || !items ? (
        <LoadingRows rows={6} />
      ) : (
        <RankedBars
          items={items}
          identityColors={colors}
          onSelect={onSelect}
          formatLabel={formatLabel}
        />
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
