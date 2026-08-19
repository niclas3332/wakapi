import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { appUrl } from '@/api/client'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, TextInput, Toggle } from '@/components/ui/Form'

export function ApiKeysTab({ settings }: { settings: SettingsResponse }) {
  return (
    <>
      <KeyListCard settings={settings} />
      <NewKeyCard />
    </>
  )
}

/** A key is a secret: hidden by default, revealed on request, copyable either way. */
function KeyValue({ value }: { value: string }) {
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)

  return (
    <div className="flex flex-wrap items-center gap-2">
      <code className="min-w-0 flex-1 truncate rounded border border-ink-700 bg-ink-900 px-3 py-1.5 text-xs text-fg-muted">
        {revealed ? value : '•'.repeat(Math.min(value.length, 36))}
      </code>
      <Button onClick={() => setRevealed((current) => !current)}>
        {revealed ? 'Hide' : 'Show'}
      </Button>
      <Button
        onClick={() =>
          navigator.clipboard.writeText(value).then(
            () => setCopied(true),
            () => setCopied(false),
          )
        }
      >
        {copied ? 'Copied' : 'Copy'}
      </Button>
    </div>
  )
}

function KeyListCard({ settings }: { settings: SettingsResponse }) {
  const reset = useSettingsAction(settingsApi.resetApiKey)
  const remove = useSettingsAction(settingsApi.deleteApiKey)
  const [confirmingReset, setConfirmingReset] = useState(false)

  return (
    <Card title="API keys">
      <p className="text-sm text-fg-muted">
        Your editor plugins authenticate with these. See{' '}
        <a href={appUrl('/setup')} className="text-accent hover:text-fg">
          how to set up an editor
        </a>
        .
      </p>

      <ul className="mt-3 space-y-4">
        {settings.api_keys.map((key) => (
          <li key={key.api_key}>
            <div className="mb-1.5 flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium text-fg">{key.label}</span>
              {key.read_only && <span className="eyebrow">read only</span>}
              {key.primary ? (
                <span className="eyebrow">main key</span>
              ) : (
                <Button
                  variant="danger"
                  className="ml-auto"
                  disabled={remove.state === 'pending'}
                  onClick={() => remove.run(key.api_key)}
                >
                  Delete
                </Button>
              )}
            </div>
            <KeyValue value={key.api_key} />
          </li>
        ))}
      </ul>

      <FormActions
        status={
          <>
            <ActionStatus state={reset.state} message={reset.message} />
            <ActionStatus state={remove.state} message={remove.message} />
          </>
        }
      >
        {confirmingReset ? (
          <>
            <span className="text-xs text-fg-muted">
              Every editor plugin stops working until you paste the new key.
            </span>
            <Button onClick={() => setConfirmingReset(false)}>Cancel</Button>
            <Button
              variant="danger"
              disabled={reset.state === 'pending'}
              onClick={() => {
                setConfirmingReset(false)
                reset.run(undefined as never)
              }}
            >
              Regenerate anyway
            </Button>
          </>
        ) : (
          <Button onClick={() => setConfirmingReset(true)}>Regenerate main key</Button>
        )}
      </FormActions>
    </Card>
  )
}

function NewKeyCard() {
  const [label, setLabel] = useState('')
  const [readOnly, setReadOnly] = useState(false)

  const create = useSettingsAction(
    async (input: { label: string; read_only: boolean }) => {
      const key = await settingsApi.createApiKey(input)
      return { ...key, message: `Created “${key.label}”.` }
    },
    { onSuccess: () => setLabel('') },
  )

  return (
    <Card title="Add an API key">
      <p className="text-sm text-fg-muted">
        A separate key for a script or a second machine, so you can revoke it on its own
        without touching your editors.
      </p>

      <div className="mt-3 max-w-md">
        <Field label="Name">
          {(id) => (
            <TextInput
              id={id}
              value={label}
              placeholder="CI pipeline"
              onChange={(event) => setLabel(event.target.value)}
            />
          )}
        </Field>
      </div>

      <div className="mt-2">
        <Toggle
          label="Read only"
          hint="Can read your statistics but not submit heartbeats."
          checked={readOnly}
          onChange={setReadOnly}
        />
      </div>

      <FormActions status={<ActionStatus state={create.state} message={create.message} />}>
        <Button
          variant="primary"
          disabled={!label.trim() || create.state === 'pending'}
          onClick={() => create.run({ label: label.trim(), read_only: readOnly })}
        >
          Create key
        </Button>
      </FormActions>
    </Card>
  )
}
