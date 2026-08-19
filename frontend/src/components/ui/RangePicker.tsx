import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router'

import { INTERVAL_OPTIONS, currentInterval, intervalLabel, writeIntervalCookie } from '@/lib/interval'

/**
 * Range selector for the dashboard.
 *
 * Keeps the single-key shortcuts the legacy picker had (D, E, W, M, Y, 1-4, A) and
 * keeps writing the interval cookie, so the chosen range still survives a visit.
 */
export function RangePicker() {
  const [search, setSearch] = useSearchParams()
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const from = search.get('from') ?? ''
  const to = search.get('to') ?? ''
  const interval = currentInterval(search)
  const label = intervalLabel(interval, from || undefined, to || undefined)

  function applyInterval(value: string) {
    const next = new URLSearchParams(search)
    next.set('interval', value)
    next.delete('from')
    next.delete('to')
    writeIntervalCookie(value)
    setSearch(next)
    setOpen(false)
  }

  function applyCustomRange(nextFrom: string, nextTo: string) {
    const next = new URLSearchParams(search)
    next.set('from', nextFrom)
    next.set('to', nextTo)
    next.delete('interval')
    setSearch(next)
    setOpen(false)
  }

  // Shortcuts stay out of the way while the user is typing in a field.
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey) return
      const target = event.target as HTMLElement | null
      if (target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) return

      const option = INTERVAL_OPTIONS.find((o) => o.hotkey === event.key.toLowerCase())
      if (option) {
        event.preventDefault()
        applyInterval(option.value)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  })

  useEffect(() => {
    if (!open) return
    function onClick(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    window.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className="relative" ref={containerRef}>
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        className="flex items-center gap-2 rounded border border-ink-700 bg-ink-850 px-3 py-1.5 text-sm text-fg transition-colors hover:border-ink-600"
        aria-expanded={open}
        aria-haspopup="true"
      >
        <span>{label}</span>
        <span aria-hidden="true" className="text-fg-subtle">
          ▾
        </span>
      </button>

      {open && (
        <div className="absolute right-0 z-30 mt-2 w-60 rounded-lg border border-ink-700 bg-ink-850 p-1 shadow-xl">
          <ul>
            {INTERVAL_OPTIONS.map((option) => (
              <li key={option.value}>
                <button
                  type="button"
                  onClick={() => applyInterval(option.value)}
                  className={
                    'flex w-full items-center justify-between rounded px-3 py-1.5 text-left text-sm transition-colors hover:bg-ink-800 ' +
                    (option.value === interval ? 'text-accent' : 'text-fg')
                  }
                >
                  {option.label}
                  <kbd className="tabular rounded border border-ink-700 px-1 text-[10px] text-fg-subtle">
                    {option.hotkey.toUpperCase()}
                  </kbd>
                </button>
              </li>
            ))}
          </ul>

          <form
            className="mt-1 border-t border-ink-700 p-2"
            onSubmit={(event) => {
              event.preventDefault()
              const data = new FormData(event.currentTarget)
              const nextFrom = String(data.get('from') ?? '')
              const nextTo = String(data.get('to') ?? '')
              if (nextFrom && nextTo) applyCustomRange(nextFrom, nextTo)
            }}
          >
            <label className="eyebrow block" htmlFor="range-from">
              Start
            </label>
            <input
              id="range-from"
              name="from"
              type="date"
              defaultValue={from}
              required
              className="mb-2 w-full rounded border border-ink-700 bg-ink-900 px-2 py-1 text-sm text-fg"
            />
            <label className="eyebrow block" htmlFor="range-to">
              End
            </label>
            <input
              id="range-to"
              name="to"
              type="date"
              defaultValue={to}
              required
              className="mb-2 w-full rounded border border-ink-700 bg-ink-900 px-2 py-1 text-sm text-fg"
            />
            <button
              type="submit"
              className="w-full rounded bg-accent-strong px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent"
            >
              Apply range
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
