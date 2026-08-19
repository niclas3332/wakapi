import { useEffect, type ReactNode } from 'react'
import { useSearchParams } from 'react-router'
import clsx from 'clsx'

import { useWrapped, useWrappedYears } from '@/api/queries'
import type { SummaryItem, WrappedResponse } from '@/api/types'
import { colorFor } from '@/lib/colors'
import {
  formatDate,
  formatDuration,
  formatDurationCompact,
  formatPercent,
} from '@/lib/format'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/State'
import { Reveal } from '@/components/ui/Reveal'
import { Heatmap } from '@/components/charts/Heatmap'
import { WeekdayBars } from '@/components/charts/WeekdayBars'
import { SplitBar, aiHumanParts } from '@/components/charts/SplitBar'

const WEEKDAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

const daysInYear = (year: number) => (new Date(year, 1, 29).getMonth() === 1 ? 366 : 365)

/**
 * Whether a year was tracked end to end.
 *
 * The range starts at the first heartbeat, so the year someone signed up in covers only
 * part of the calendar - 36 days, in the case that prompted this. Anything that reads
 * as a year-on-year change has to know the difference.
 */
const wasFullYear = (year: { year: number; day_count: number }) =>
  year.day_count >= daysInYear(year.year)

/**
 * A finished year told top to bottom rather than laid out in a grid.
 *
 * Every other screen in this app answers "what are my numbers" and is built for
 * scanning: dense cards, many figures at once. This one answers "what was my year",
 * which is a different question and reads badly in the same format. One statement per
 * band, the number at a size the rest of the app never uses, and the supporting detail
 * underneath it - so it is read in sequence instead of scanned.
 *
 * Only finished years appear. The server does not serve a year that is still running,
 * and the picker does not offer one: half a year has no final total, no settled average
 * and nothing meaningful to compare against the year before. The dashboard is where the
 * year so far lives.
 */
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
  // The year before, whole against whole. A look back that cannot say "more than last
  // year" is just a filtered dashboard.
  const previousQuery = useWrapped(years.includes(year - 1) ? year - 1 : undefined)

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
    <div className="mx-auto max-w-3xl">
      <div className="flex flex-wrap items-baseline gap-3 pb-2">
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
                  'tabular rounded px-3 py-1.5 text-sm transition-colors',
                  option === year
                    ? 'bg-accent-strong text-white'
                    : 'text-fg-muted hover:bg-ink-800 hover:text-fg',
                )}
              >
                {option}
              </button>
            ))}
          </div>
        )}
      </div>

      {!yearsQuery.isPending && years.length === 0 && (
        <EmptyState message="No finished year to look back on yet. Your first wrapped arrives in January — until then the dashboard has the year so far." />
      )}

      {query.isError && (
        <ErrorState
          message={`Could not load ${year}: ${(query.error as Error).message}`}
          onRetry={() => query.refetch()}
        />
      )}

      {(query.isPending || !data) && years.length > 0 && !query.isError && (
        <div className="space-y-10 py-10">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-32 w-2/3" />
          <Skeleton className="h-32 w-2/3" />
        </div>
      )}

      {data && data.total_seconds === 0 && (
        <EmptyState message={`Nothing tracked in ${data.year}.`} />
      )}

      {data && data.total_seconds > 0 && (
        <div className="divide-y divide-ink-800">
          <Opening data={data} previous={previousQuery.data} />
          <Showing data={data} />
          <Streak data={data} />
          <BusiestDay data={data} />
          <TheYear data={data} />
          <TheWeek data={data} />
          <Podium data={data} />
          <Machine data={data} />
        </div>
      )}
    </div>
  )
}

/** One band of the story: a label, one number at full size, and a line of context. */
function Band({
  label,
  children,
  footnote,
}: {
  label: string
  children: ReactNode
  footnote?: ReactNode
}) {
  return (
    <Reveal className="py-12 first:pt-6">
      <p className="eyebrow">{label}</p>
      <div className="mt-3">{children}</div>
      {footnote && <p className="mt-4 max-w-prose text-sm text-fg-muted">{footnote}</p>}
    </Reveal>
  )
}

/**
 * The headline figure. Sized in viewport units so it stays the biggest thing on the
 * screen at any width, with a ceiling so it does not run off a wide monitor.
 */
function Big({ children, accent = false }: { children: ReactNode; accent?: boolean }) {
  return (
    <p
      className={clsx(
        'tabular font-bold leading-[0.95] tracking-tight',
        accent ? 'text-accent' : 'text-fg',
      )}
      style={{ fontSize: 'clamp(2.75rem, 11vw, 5.5rem)' }}
    >
      {children}
    </p>
  )
}

function Opening({ data, previous }: { data: WrappedResponse; previous?: WrappedResponse }) {
  const hours = Math.round(data.total_seconds / 3600)

  let comparison: ReactNode = null
  if (previous && previous.total_seconds > 0) {
    const previousHours = Math.round(previous.total_seconds / 3600)

    if (!wasFullYear(previous)) {
      // Comparing a whole year against a part of one produces a huge percentage that
      // measures when tracking started, not how much was worked.
      comparison = (
        <>
          You tracked {previousHours} hours in {previous.year}, but only {previous.day_count}{' '}
          days of it — too different to compare.
        </>
      )
    } else {
      const change = (data.total_seconds - previous.total_seconds) / previous.total_seconds
      comparison =
        Math.abs(change) < 0.01 ? (
          <>
            Almost exactly as much as {previous.year}, when you coded {previousHours} hours.
          </>
        ) : (
          <>
            <span className="text-fg">
              {formatPercent(Math.abs(change))} {change > 0 ? 'more' : 'less'}
            </span>{' '}
            than {previous.year}, when you coded {previousHours} hours.
          </>
        )
    }
  }

  return (
    <Reveal className="pt-6 pb-12">
      <p className="eyebrow">Your year in code</p>
      <p
        className="tabular font-bold leading-none tracking-tighter text-accent"
        style={{ fontSize: 'clamp(4rem, 20vw, 9rem)' }}
      >
        {data.year}
      </p>
      <p className="mt-4 text-xl text-fg sm:text-2xl">
        You spent{' '}
        <span className="tabular font-semibold">{hours.toLocaleString()} hours</span> in your
        editor.
      </p>
      {comparison && <p className="mt-2 max-w-prose text-sm text-fg-muted">{comparison}</p>}
    </Reveal>
  )
}

function Showing({ data }: { data: WrappedResponse }) {
  const share = data.day_count > 0 ? data.active_days / data.day_count : 0

  // The range starts at the first heartbeat, so it is only the whole year for a year
  // that was tracked end to end. Saying "of the 36 days in 2023" would be plainly
  // wrong; naming what the count actually covers is not.
  const partial = !wasFullYear(data)

  return (
    <Band
      label="Days you showed up"
      footnote={
        <>
          That is {formatPercent(share)} of the {data.day_count} days{' '}
          {partial ? 'you were tracking' : `in ${data.year}`}, and an average of{' '}
          {formatDuration(data.daily_average)} across every one of them — the quiet days
          included.
        </>
      }
    >
      <Big>
        {data.active_days}
        <span className="text-fg-subtle"> / {data.day_count}</span>
      </Big>
    </Band>
  )
}

function Streak({ data }: { data: WrappedResponse }) {
  if (data.streak.longest <= 1) return null

  return (
    <Band
      label="Longest streak"
      footnote={
        data.streak.longest_start && data.streak.longest_end ? (
          <>
            From {formatDate(data.streak.longest_start)} to{' '}
            {formatDate(data.streak.longest_end)}, without missing a day.
          </>
        ) : undefined
      }
    >
      <Big>
        {data.streak.longest}
        <span className="ml-3 text-2xl font-medium text-fg-muted sm:text-3xl">days in a row</span>
      </Big>
    </Band>
  )
}

function BusiestDay({ data }: { data: WrappedResponse }) {
  if (!data.most_active_day) return null

  const date = new Date(data.most_active_day.date)
  return (
    <Band
      label="Your busiest day"
      footnote={
        <>
          {WEEKDAY_NAMES[date.getDay()]}, {formatDate(data.most_active_day.date)}. That is{' '}
          {(data.most_active_day.seconds / data.daily_average).toFixed(1)}× an average day.
        </>
      }
    >
      <Big>{formatDurationCompact(data.most_active_day.seconds)}</Big>
    </Band>
  )
}

function TheYear({ data }: { data: WrappedResponse }) {
  // No footnote here: the heatmap carries its own legend, and repeating it underneath
  // would say the same thing twice.
  return (
    <Band label="Every day of it">
      <Heatmap days={data.daily} />
    </Band>
  )
}

function TheWeek({ data }: { data: WrappedResponse }) {
  const busiest = data.weekdays.reduce((best, day) =>
    day.average_seconds > best.average_seconds ? day : best,
  )

  return (
    <Band
      label="Your week"
      footnote={
        busiest.average_seconds > 0 ? (
          <>
            {WEEKDAY_NAMES[busiest.weekday]} was your day, at{' '}
            {formatDuration(busiest.average_seconds)} on average.
          </>
        ) : undefined
      }
    >
      <WeekdayBars weekdays={data.weekdays} />
    </Band>
  )
}

/** The three #1s, each with its runners-up underneath. */
function Podium({ data }: { data: WrappedResponse }) {
  const categories: { label: string; items: SummaryItem[]; colors?: Record<string, string> }[] = [
    { label: 'Project', items: data.projects },
    { label: 'Language', items: data.languages, colors: data.language_colors },
    { label: 'Editor', items: data.editors, colors: data.editor_colors },
  ]

  return (
    <Band label="Your number ones">
      <div className="grid gap-8 sm:grid-cols-3">
        {categories.map((category) => {
          const [first, ...rest] = category.items
          if (!first) return null
          return (
            <div key={category.label} className="min-w-0">
              <p className="eyebrow">{category.label}</p>
              <p className="mt-1 truncate text-2xl font-semibold text-fg" title={first.key}>
                {first.key}
              </p>
              <p className="tabular mt-0.5 text-sm text-fg-muted">
                {formatDuration(first.seconds)}
              </p>
              {rest.length > 0 && (
                <ol className="mt-3 space-y-1 border-t border-ink-800 pt-3">
                  {rest.slice(0, 4).map((item, index) => (
                    <li key={item.key} className="flex items-baseline gap-2 text-xs">
                      <span className="tabular w-3 shrink-0 text-fg-subtle">{index + 2}</span>
                      <span
                        aria-hidden="true"
                        className="inline-block h-2 w-2 shrink-0 rounded-sm"
                        style={{
                          backgroundColor: colorFor(item.key, index + 1, category.colors),
                        }}
                      />
                      <span className="truncate text-fg-muted" title={item.key}>
                        {item.key}
                      </span>
                      <span className="tabular ml-auto shrink-0 text-fg-subtle">
                        {formatDurationCompact(item.seconds)}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </div>
          )
        })}
      </div>
    </Band>
  )
}

function Machine({ data }: { data: WrappedResponse }) {
  const total = data.ai_seconds + data.human_seconds
  if (total <= 0) return null

  return (
    <Band
      label="You and the machine"
      footnote={
        <>
          {formatPercent(data.ai_seconds / total)} of your coding time in {data.year} was in the
          AI category. The rest you typed yourself.
        </>
      }
    >
      <div className="max-w-md">
        <SplitBar parts={aiHumanParts(data.ai_seconds, data.human_seconds, formatDuration)} />
      </div>
    </Band>
  )
}
