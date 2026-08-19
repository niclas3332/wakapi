import { useEffect, useMemo, useState } from 'react'

import { settingsApi, useAIPrices, useSettingsAction } from '@/api/settings'
import type { AIPriceTable } from '@/api/types'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, FormActions, TextInput } from '@/components/ui/Form'
import { ErrorState, LoadingRows } from '@/components/ui/State'

/**
 * One editable row. The stored shape is model → version → price, but nesting is
 * miserable to edit, so the form works on a flat list and rebuilds the nesting on save.
 */
type PriceRow = {
  id: number
  model: string
  version: string
  input: string
  cachedInput: string
  output: string
}

let nextId = 0

function flatten(table: AIPriceTable): PriceRow[] {
  const rows: PriceRow[] = []
  for (const [model, versions] of Object.entries(table)) {
    for (const [version, price] of Object.entries(versions)) {
      rows.push({
        id: nextId++,
        model,
        version,
        input: String(price.input ?? 0),
        cachedInput: String(price.cached_input ?? 0),
        output: String(price.output ?? 0),
      })
    }
  }
  return rows.sort((a, b) => a.model.localeCompare(b.model) || a.version.localeCompare(b.version))
}

function rebuild(rows: PriceRow[]): AIPriceTable {
  const table: AIPriceTable = {}
  for (const row of rows) {
    const model = row.model.trim().toLowerCase()
    const version = row.version.trim()
    if (!model || !version) continue
    table[model] ??= {}
    table[model][version] = {
      input: Number(row.input) || 0,
      cached_input: Number(row.cachedInput) || 0,
      output: Number(row.output) || 0,
    }
  }
  return table
}

export function AiPricesCard() {
  const query = useAIPrices(true)
  const [rows, setRows] = useState<PriceRow[] | null>(null)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    if (query.data && !rows) setRows(flatten(query.data.prices))
  }, [query.data, rows])

  const action = useSettingsAction(settingsApi.updateAIPrices, { invalidate: [['ai-prices']] })
  const reset = useSettingsAction(settingsApi.resetAIPrices, {
    invalidate: [['ai-prices']],
    // Drop the local draft so the form reloads from the server rather than showing the
    // edits that were just discarded.
    onSuccess: () => setRows(null),
  })

  const visible = useMemo(() => {
    if (!rows) return []
    const needle = filter.trim().toLowerCase()
    if (!needle) return rows
    return rows.filter((row) => `${row.model} ${row.version}`.toLowerCase().includes(needle))
  }, [rows, filter])

  if (query.isError) {
    return (
      <ErrorState
        message={`Could not load the price table: ${(query.error as Error).message}`}
        onRetry={() => query.refetch()}
      />
    )
  }
  if (query.isPending || !rows || !query.data) {
    return (
      <Card title="AI model prices">
        <LoadingRows rows={6} />
      </Card>
    )
  }

  const update = (id: number, patch: Partial<PriceRow>) =>
    setRows((current) => current?.map((row) => (row.id === id ? { ...row, ...patch } : row)) ?? null)

  return (
    <Card
      title="AI model prices"
      meta={query.data.customized ? 'customised' : 'built-in defaults'}
    >
      <p className="text-sm text-fg-muted">
        Cost per million tokens, used to price the AI screens. A model with no entry
        contributes nothing to the cost and is reported as unpriced rather than free.
        When a heartbeat's version cannot be determined, the model's newest entry
        applies; when no cache price is set, the input price is used.
      </p>

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <TextInput
          type="search"
          value={filter}
          placeholder="Filter by model"
          aria-label="Filter by model"
          onChange={(event) => setFilter(event.target.value)}
          className="max-w-56"
        />
        <span className="text-xs text-fg-subtle">
          {visible.length} of {rows.length} entries
        </span>
      </div>

      <div className="mt-3 max-h-[28rem] overflow-auto rounded border border-ink-700">
        <table className="w-full min-w-[38rem] text-sm">
          <thead className="sticky top-0 bg-ink-850">
            <tr className="border-b border-ink-700 text-left">
              <th className="eyebrow px-2 py-2 font-normal">Model</th>
              <th className="eyebrow px-2 py-2 font-normal">Version</th>
              <th className="eyebrow px-2 py-2 font-normal">Input</th>
              <th className="eyebrow px-2 py-2 font-normal">Cache read</th>
              <th className="eyebrow px-2 py-2 font-normal">Output</th>
              <th className="px-2 py-2" />
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-700">
            {visible.map((row) => (
              <tr key={row.id}>
                <td className="px-2 py-1.5">
                  <TextInput
                    value={row.model}
                    aria-label="Model"
                    onChange={(event) => update(row.id, { model: event.target.value })}
                  />
                </td>
                <td className="px-2 py-1.5">
                  <TextInput
                    value={row.version}
                    aria-label="Version"
                    onChange={(event) => update(row.id, { version: event.target.value })}
                  />
                </td>
                {(['input', 'cachedInput', 'output'] as const).map((field) => (
                  <td key={field} className="px-2 py-1.5">
                    <TextInput
                      type="number"
                      min={0}
                      step="0.01"
                      value={row[field]}
                      aria-label={field}
                      onChange={(event) => update(row.id, { [field]: event.target.value })}
                      className="tabular"
                    />
                  </td>
                ))}
                <td className="px-2 py-1.5 text-right">
                  <Button
                    variant="danger"
                    onClick={() => setRows((current) => current?.filter((r) => r.id !== row.id) ?? null)}
                  >
                    Remove
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <FormActions
        status={
          <>
            <ActionStatus state={action.state} message={action.message} />
            <ActionStatus state={reset.state} message={reset.message} />
          </>
        }
      >
        <Button
          onClick={() =>
            setRows((current) => [
              { id: nextId++, model: '', version: '', input: '0', cachedInput: '0', output: '0' },
              ...(current ?? []),
            ])
          }
        >
          Add model
        </Button>
        {/*
          Discards the stored table rather than saving a copy of the defaults, so a
          later wakapi update carrying new rates still reaches this instance.
        */}
        <Button
          disabled={!query.data.customized || reset.state === 'pending'}
          onClick={() => reset.run(undefined as never)}
        >
          Use built-in prices
        </Button>
        <Button
          variant="primary"
          disabled={action.state === 'pending'}
          onClick={() => action.run(rebuild(rows))}
        >
          Save prices
        </Button>
      </FormActions>
    </Card>
  )
}
