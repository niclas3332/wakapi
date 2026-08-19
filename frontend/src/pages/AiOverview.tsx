import { useMemo } from 'react'
import { useSearchParams } from 'react-router'

import { useAIFiles, useAIOverview } from '@/api/queries'
import type { AICost, AIFile, AILines, AIOverviewResponse, AITokens } from '@/api/types'
import { toApiParams } from '@/lib/interval'
import { fileName } from '@/lib/entities'
import { AI_COLOR, HUMAN_COLOR } from '@/lib/colors'
import {
  formatCost,
  formatCount,
  formatDate,
  formatDuration,
  formatDurationCompact,
  formatTokens,
} from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingRows, Skeleton } from '@/components/ui/State'
import { RangePicker } from '@/components/ui/RangePicker'
import { StatTile } from '@/components/ui/StatTile'
import { SplitBar, aiHumanParts } from '@/components/charts/SplitBar'
import { RankedBars } from '@/components/charts/RankedBars'

/** Lines written, counting both additions and removals as work done. */
function lineVolume(lines: AILines, side: 'ai' | 'human'): number {
  return side === 'ai'
    ? lines.ai_added + lines.ai_removed
    : lines.human_added + lines.human_removed
}

export function AiOverviewPage() {
  const [search] = useSearchParams()
  const params = useMemo(() => toApiParams(search), [search])

  const query = useAIOverview(params)
  const aiFiles = useAIFiles(params, 'ai')
  const humanFiles = useAIFiles(params, 'human')

  if (query.isError) {
    return (
      <ErrorState
        message={`Could not load your AI statistics: ${(query.error as Error).message}`}
        onRetry={() => query.refetch()}
      />
    )
  }

  const data = query.data

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">AI</h1>
        <div className="ml-auto">
          <RangePicker />
        </div>
      </div>

      {query.isPending || !data ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-[4.75rem]" />
          ))}
        </div>
      ) : data.totals.tokens.total === 0 && data.ai_seconds === 0 ? (
        <Card>
          <EmptyState message="No AI activity in this range. It appears here once a plugin reports work in the “ai coding” category." />
        </Card>
      ) : (
        <>
          <Accuracy data={data} />

          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
            <StatTile
              label="AI time"
              value={formatDurationCompact(data.ai_seconds)}
              note={
                data.ai_seconds + data.human_seconds > 0
                  ? `${Math.round((data.ai_seconds / (data.ai_seconds + data.human_seconds)) * 100)}% of coding`
                  : undefined
              }
              emphasis
            />
            <StatTile
              label="Cost"
              value={
                data.totals.cost_tracked.priced
                  ? formatCost(data.totals.cost_tracked.total)
                  : data.totals.cost.priced
                    ? formatCost(data.totals.cost.total)
                    : '—'
              }
              note={data.cached_tokens_since ? 'measured days only' : 'upper bound'}
            />
            <StatTile
              label="Tokens"
              value={formatTokens(data.totals.tokens.total)}
              note={`${formatTokens(data.totals.tokens.output)} written`}
            />
            <StatTile
              label="AI lines"
              value={formatCount(lineVolume(data.totals.lines, 'ai'))}
              note={`${formatCount(lineVolume(data.totals.lines, 'human'))} by hand`}
            />
            <StatTile
              label="Prompts"
              value={formatCount(data.totals.prompt_count)}
              note={
                data.totals.prompt_count > 0
                  ? `${formatCount(Math.round(data.totals.prompt_length_total / data.totals.prompt_count))} chars each`
                  : undefined
              }
            />
            <StatTile label="Sessions" value={formatCount(data.totals.session_count)} note="distinct" />
          </div>

          <div className="grid items-start gap-4 lg:grid-cols-3">
            <Card title="Time" meta="AI against hand-written">
              <SplitBar parts={aiHumanParts(data.ai_seconds, data.human_seconds, formatDuration)} />
            </Card>

            <Card title="Lines" meta="added and removed">
              <SplitBar
                parts={aiHumanParts(
                  lineVolume(data.totals.lines, 'ai'),
                  lineVolume(data.totals.lines, 'human'),
                  formatCount,
                )}
              />
            </Card>

            <Card title="Tokens" meta="what the prompts cost">
              <SplitBar
                parts={[
                  {
                    label: 'Cache reads',
                    value: data.totals.tokens.cached_input,
                    color: 'var(--color-series-1)',
                    display: formatTokens(data.totals.tokens.cached_input),
                  },
                  {
                    label: 'Fresh input',
                    value: data.totals.tokens.input,
                    color: 'var(--color-series-4)',
                    display: formatTokens(data.totals.tokens.input),
                  },
                  {
                    label: 'Output',
                    value: data.totals.tokens.output,
                    color: 'var(--color-series-3)',
                    display: formatTokens(data.totals.tokens.output),
                  },
                ]}
              />
            </Card>
          </div>

          <Card title="By model" meta={`${data.by_model.length} in this range`}>
            <ModelTable data={data} />
          </Card>

          <Card title="By day" meta="AI against hand-written lines">
            <DailyLines data={data} />
          </Card>

          <div className="grid items-start gap-4 lg:grid-cols-2">
            <Card title="AI-assisted projects" meta="lines changed">
              <RankedBars
                items={data.by_project
                  .map((project) => ({
                    key: project.project || 'unknown',
                    seconds: lineVolume(project.lines, 'ai'),
                  }))
                  .filter((item) => item.seconds > 0)}
                emptyMessage="No AI line changes recorded for any project."
                formatValue={formatCount}
              />
            </Card>
            <Card title="Hand-written projects" meta="lines changed">
              <RankedBars
                items={data.by_project
                  .map((project) => ({
                    key: project.project || 'unknown',
                    seconds: lineVolume(project.lines, 'human'),
                  }))
                  .filter((item) => item.seconds > 0)}
                emptyMessage="No hand-written line changes recorded for any project."
                formatValue={formatCount}
              />
            </Card>
          </div>

          <div className="grid items-start gap-4 lg:grid-cols-2">
            <FilesCard
              title="AI-assisted files"
              meta="lines changed"
              query={aiFiles}
              value={(file) => file.ai_lines}
            />
            <FilesCard
              title="Hand-edited files"
              meta="lines changed"
              query={humanFiles}
              value={(file) => Math.abs(file.human_lines)}
            />
          </div>
        </>
      )}
    </div>
  )
}

/**
 * What the numbers on this page can and cannot say.
 *
 * Shown above the figures rather than under them: a cost that excludes most of its
 * tokens is worse than no cost at all if the reader does not know it.
 */
function Accuracy({ data }: { data: AIOverviewResponse }) {
  const notes: string[] = []

  if (!data.cached_tokens_since) {
    notes.push(
      'Cache tokens have not been recorded yet. Until an editor sends them, input counts and costs are whatever the plugin reported — for older data that folded cache reads into the input figure, which prices them about ten times too high.',
    )
  } else {
    notes.push(
      `Cache tokens were first recorded on ${formatDate(data.cached_tokens_since)}. Costs before that are an upper bound; the headline figure covers only the days that measured them.`,
    )
  }

  if (data.totals.unpriced_tokens > 0) {
    const exact = (data.totals.unpriced_tokens / data.totals.tokens.total) * 100
    // Rounding a real share down to "0%" reads as "none", which is the opposite of
    // what this note is for.
    const share = exact < 1 ? 'under 1' : Math.round(exact).toString()
    notes.push(
      `${formatTokens(data.totals.unpriced_tokens)} tokens (${share}%) come from models with no price configured and cost nothing here. Some plugins report AI usage without naming a model.`,
    )
  }

  if (!data.prices_configured) {
    notes.push('Prices are the built-in defaults. An administrator can change them under Settings → Admin.')
  }

  return (
    <div className="rounded-lg border border-ink-700 bg-ink-850 px-4 py-3">
      <p className="eyebrow mb-1.5">How exact this is</p>
      <ul className="space-y-1 text-xs text-fg-muted">
        {notes.map((note) => (
          <li key={note}>{note}</li>
        ))}
      </ul>
    </div>
  )
}

function ModelTable({ data }: { data: AIOverviewResponse }) {
  if (data.by_model.length === 0) {
    return <EmptyState message="No model reported any tokens in this range." />
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[44rem] text-sm">
        <thead>
          <tr className="border-b border-ink-700 text-left">
            <th className="eyebrow pb-2 font-normal">Model</th>
            <th className="eyebrow pb-2 text-right font-normal">Cache</th>
            <th className="eyebrow pb-2 text-right font-normal">Input</th>
            <th className="eyebrow pb-2 text-right font-normal">Output</th>
            <th className="eyebrow pb-2 text-right font-normal">Lines</th>
            <th className="eyebrow pb-2 text-right font-normal">Prompts</th>
            <th className="eyebrow pb-2 text-right font-normal">Cost</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-ink-700">
          {data.by_model.map((model) => (
            <tr key={`${model.model}-${model.version}`}>
              <td className="py-2 pr-3">
                <span className="text-fg">{model.model}</span>
                {model.version && <span className="ml-1.5 text-fg-subtle">{model.version}</span>}
              </td>
              <TokenCell tokens={model.tokens} field="cached_input" />
              <TokenCell tokens={model.tokens} field="input" />
              <TokenCell tokens={model.tokens} field="output" />
              <td className="tabular py-2 text-right text-fg-muted">
                {formatCount(lineVolume(model.lines, 'ai'))}
              </td>
              <td className="tabular py-2 text-right text-fg-muted">
                {formatCount(model.prompt_count)}
              </td>
              <CostCell cost={model.cost} />
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function TokenCell({ tokens, field }: { tokens: AITokens; field: keyof AITokens }) {
  return <td className="tabular py-2 text-right text-fg-muted">{formatTokens(tokens[field])}</td>
}

function CostCell({ cost }: { cost: AICost }) {
  if (!cost.priced) {
    return (
      <td className="tabular py-2 text-right text-fg-subtle" title="No price configured for this model">
        no price
      </td>
    )
  }
  return <td className="tabular py-2 text-right text-fg">{formatCost(cost.total)}</td>
}

/** One stacked column per day: AI lines against hand-written lines. */
function DailyLines({ data }: { data: AIOverviewResponse }) {
  const days = data.by_day.filter(
    (day) => lineVolume(day.lines, 'ai') > 0 || lineVolume(day.lines, 'human') > 0,
  )
  if (days.length === 0) {
    return <EmptyState message="No line changes recorded in this range." />
  }

  const max = days.reduce(
    (widest, day) => Math.max(widest, lineVolume(day.lines, 'ai') + lineVolume(day.lines, 'human')),
    0,
  )

  return (
    <div className="flex h-44 items-end gap-1 overflow-x-auto">
      {days.map((day) => {
        const ai = lineVolume(day.lines, 'ai')
        const human = lineVolume(day.lines, 'human')
        return (
          <div
            key={day.date}
            className="flex h-full min-w-2 max-w-8 flex-1 flex-col justify-end gap-[2px]"
            title={`${formatDate(day.date)}: ${formatCount(ai)} AI, ${formatCount(human)} by hand`}
          >
            {ai > 0 && (
              <span
                className="w-full rounded-[2px]"
                style={{ height: `${(ai / max) * 100}%`, backgroundColor: AI_COLOR }}
              />
            )}
            {human > 0 && (
              <span
                className="w-full rounded-[2px]"
                style={{ height: `${(human / max) * 100}%`, backgroundColor: HUMAN_COLOR }}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

function FilesCard({
  title,
  meta,
  query,
  value,
}: {
  title: string
  meta: string
  query: ReturnType<typeof useAIFiles>
  value: (file: AIFile) => number
}) {
  return (
    <Card title={title} meta={meta}>
      {query.isPending ? (
        <LoadingRows rows={6} />
      ) : query.isError ? (
        <ErrorState message="Could not load the file breakdown." onRetry={() => query.refetch()} />
      ) : (
        <>
          <RankedBars
            items={(query.data?.items ?? [])
              .map((file) => ({ key: file.entity, seconds: value(file) }))
              .filter((item) => item.seconds > 0)}
            emptyMessage="No line changes recorded for any file."
            formatLabel={fileName}
            formatValue={formatCount}
          />
          {query.data?.truncated && (
            <p className="mt-3 text-xs text-fg-subtle">
              Showing the top {query.data.limit}. Files are counted on demand, not
              pre-aggregated.
            </p>
          )}
        </>
      )}
    </Card>
  )
}
