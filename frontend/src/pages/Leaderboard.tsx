import { useSearchParams } from 'react-router'
import clsx from 'clsx'

import { useLeaderboard } from '@/api/queries'
import { appUrl } from '@/api/client'
import { formatDuration } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingRows } from '@/components/ui/State'

/** Medal tints for the first three places, matching the legacy page. */
const RANK_TINT = ['text-[#e0b64a]', 'text-[#b8c1cc]', 'text-[#c07d50]']

export function LeaderboardPage() {
  const [search, setSearch] = useSearchParams()
  const by = search.get('by') ?? ''
  const key = search.get('key') ?? ''

  const query = useLeaderboard(by, key)

  function setParam(name: string, value: string) {
    const next = new URLSearchParams(search)
    if (value) next.set(name, value)
    else next.delete(name)
    if (name === 'by') next.delete('key')
    setSearch(next)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold text-fg">Leaderboard</h1>
        {query.data && <span className="eyebrow">{query.data.interval_label}</span>}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex rounded border border-ink-700 p-0.5">
          {[
            { value: '', label: 'Total' },
            { value: 'language', label: 'By language' },
          ].map((tab) => (
            <button
              key={tab.value}
              type="button"
              onClick={() => setParam('by', tab.value)}
              className={clsx(
                'rounded px-3 py-1 text-sm transition-colors',
                by === tab.value ? 'bg-ink-800 text-fg' : 'text-fg-muted hover:text-fg',
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {by === 'language' &&
          query.data?.top_keys.map((topKey) => (
            <button
              key={topKey}
              type="button"
              onClick={() => setParam('key', topKey)}
              className={clsx(
                'rounded-full border px-2.5 py-1 text-xs transition-colors',
                topKey.toLowerCase() === query.data!.key.toLowerCase()
                  ? 'border-accent/50 bg-accent-soft text-fg'
                  : 'border-ink-700 text-fg-muted hover:border-ink-600 hover:text-fg',
              )}
            >
              {topKey}
            </button>
          ))}
      </div>

      <Card>
        {query.isPending ? (
          <LoadingRows rows={8} />
        ) : query.isError ? (
          <ErrorState
            message={`Could not load the leaderboard: ${(query.error as Error).message}`}
            onRetry={() => query.refetch()}
          />
        ) : query.data!.items.length === 0 ? (
          <EmptyState
            message="Nobody is on the leaderboard yet. Opt in from your settings to take part."
            action={
              <a
                href={appUrl('/settings#permissions')}
                className="rounded border border-ink-600 px-3 py-1.5 text-sm text-fg transition-colors hover:bg-ink-800"
              >
                Open permissions
              </a>
            }
          />
        ) : (
          <ol className="space-y-1">
            {query.data!.items.map((item) => (
              <li
                key={`${item.user_id}-${item.key ?? ''}`}
                className={clsx(
                  'flex items-center gap-3 rounded px-2 py-2',
                  item.is_me && 'bg-accent-soft ring-1 ring-accent/30',
                )}
              >
                <span
                  className={clsx(
                    'tabular w-8 shrink-0 text-right text-sm',
                    RANK_TINT[item.rank - 1] ?? 'text-fg-subtle',
                  )}
                >
                  {item.rank}
                </span>
                <span className="truncate text-sm text-fg">
                  {item.user_id}
                  {item.is_me && <span className="ml-2 text-xs text-accent">you</span>}
                </span>
                {item.languages && item.languages.length > 0 && (
                  <span className="hidden truncate text-xs text-fg-subtle sm:block">
                    {item.languages.slice(0, 3).join(' · ')}
                  </span>
                )}
                <span className="tabular ml-auto shrink-0 text-sm text-fg-muted">
                  {formatDuration(item.seconds)}
                </span>
              </li>
            ))}
          </ol>
        )}
      </Card>
    </div>
  )
}
