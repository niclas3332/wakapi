import { useSearchParams } from 'react-router'

import type { AvailableFilters } from '@/api/types'
import { activeFilters, type FilterKey } from '@/lib/interval'

/** Only the dimensions the api can offer a value list for get a dropdown. */
const SELECTABLE: { key: FilterKey; label: string; source: keyof AvailableFilters }[] = [
  { key: 'project', label: 'Project', source: 'projects' },
  { key: 'language', label: 'Language', source: 'languages' },
  { key: 'machine', label: 'Machine', source: 'machines' },
  { key: 'label', label: 'Label', source: 'labels' },
  { key: 'category', label: 'Category', source: 'categories' },
]

const LABELS: Record<FilterKey, string> = {
  project: 'Project',
  language: 'Language',
  editor: 'Editor',
  machine: 'Machine',
  operating_system: 'OS',
  label: 'Label',
  category: 'Category',
  branch: 'Branch',
  entity: 'File',
}

export function FilterBar({ filters }: { filters?: AvailableFilters }) {
  const [search, setSearch] = useSearchParams()
  const active = activeFilters(search)

  function setFilter(key: FilterKey, value: string) {
    const next = new URLSearchParams(search)
    if (value) next.set(key, value)
    else next.delete(key)
    setSearch(next)
  }

  function clearAll() {
    const next = new URLSearchParams(search)
    for (const { key } of active) next.delete(key)
    setSearch(next)
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {SELECTABLE.map(({ key, label, source }) => {
        const options = filters?.[source] ?? []
        const value = search.get(key) ?? ''
        return (
          <label key={key} className="flex items-center gap-1.5">
            <span className="sr-only">{label}</span>
            {/*
              A select sizes itself to its widest option, and a long-range view can
              hold hundreds of project names - so the width is pinned here.
            */}
            <select
              value={value}
              onChange={(event) => setFilter(key, event.target.value)}
              disabled={options.length === 0}
              className="w-40 truncate rounded border border-ink-700 bg-ink-850 px-2 py-1.5 text-xs text-fg transition-colors hover:border-ink-600 disabled:opacity-40"
            >
              <option value="">{label}: all</option>
              {options.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>
        )
      })}

      {/* Filters set from elsewhere (a chart row, a deep link) show as removable chips. */}
      {active
        .filter(({ key }) => !SELECTABLE.some((s) => s.key === key))
        .map(({ key, value }) => (
          <button
            key={key}
            type="button"
            onClick={() => setFilter(key, '')}
            className="flex items-center gap-1.5 rounded-full border border-accent/40 bg-accent-soft px-2.5 py-1 text-xs text-fg transition-colors hover:border-accent"
            title={`Remove ${LABELS[key]} filter`}
          >
            <span className="text-fg-muted">{LABELS[key]}:</span>
            <span className="max-w-40 truncate">{value}</span>
            <span aria-hidden="true" className="text-fg-subtle">
              ×
            </span>
          </button>
        ))}

      {active.length > 0 && (
        <button
          type="button"
          onClick={clearAll}
          className="text-xs text-fg-subtle transition-colors hover:text-fg"
        >
          Clear filters
        </button>
      )}
    </div>
  )
}
