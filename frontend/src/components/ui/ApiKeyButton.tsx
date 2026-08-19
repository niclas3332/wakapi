import { useEffect, useState } from 'react'

/**
 * The api key is the one thing a user needs when setting up a new editor, so it stays
 * one click away. Revealed on demand rather than printed into the page.
 */
export function ApiKeyButton({ apiKey }: { apiKey: string }) {
  const [open, setOpen] = useState(false)
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = window.setTimeout(() => setCopied(false), 2000)
    return () => window.clearTimeout(timer)
  }, [copied])

  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  async function copy() {
    try {
      await navigator.clipboard.writeText(apiKey)
      setCopied(true)
    } catch {
      // Clipboard access can be denied; revealing the key lets the user copy manually.
      setRevealed(true)
    }
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        className="rounded border border-ink-700 px-2.5 py-1.5 text-xs text-fg-muted transition-colors hover:border-ink-600 hover:text-fg"
        aria-expanded={open}
      >
        API key
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-20" onClick={() => setOpen(false)} aria-hidden="true" />
          <div className="absolute right-0 z-30 mt-2 w-80 rounded-lg border border-ink-700 bg-ink-850 p-3 shadow-xl">
            <p className="mb-2 text-xs text-fg-muted">
              Paste this into <code className="text-fg">~/.wakatime.cfg</code> as{' '}
              <code className="text-fg">api_key</code>.
            </p>
            <div className="flex items-center gap-2">
              <code className="tabular flex-1 truncate rounded border border-ink-700 bg-ink-900 px-2 py-1.5 text-xs text-fg">
                {revealed ? apiKey : '•'.repeat(Math.min(apiKey.length, 36))}
              </code>
              <button
                type="button"
                onClick={() => setRevealed((value) => !value)}
                className="rounded border border-ink-700 px-2 py-1.5 text-xs text-fg-muted transition-colors hover:text-fg"
              >
                {revealed ? 'Hide' : 'Show'}
              </button>
            </div>
            <button
              type="button"
              onClick={copy}
              className="mt-2 w-full rounded bg-accent-strong px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent"
            >
              {copied ? 'Copied' : 'Copy key'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}
