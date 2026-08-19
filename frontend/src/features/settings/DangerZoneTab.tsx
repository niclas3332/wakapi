import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { appUrl } from '@/api/client'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, TextInput } from '@/components/ui/Form'

/**
 * A destructive action asks for the thing it is about to destroy, typed out.
 *
 * A yes/no dialog is answered reflexively; typing the name is the smallest gate that
 * actually requires reading what is about to happen.
 */
function ConfirmBox({
  expect,
  label,
  hint,
  actionLabel,
  onConfirm,
  disabled,
}: {
  expect: string
  label: string
  hint: string
  actionLabel: string
  onConfirm: () => void
  disabled?: boolean
}) {
  const [typed, setTyped] = useState('')

  return (
    <div className="flex flex-wrap items-end gap-3">
      <Field label={label} hint={hint} className="min-w-56 flex-1">
        {(id) => (
          <TextInput
            id={id}
            value={typed}
            placeholder={expect}
            onChange={(event) => setTyped(event.target.value)}
          />
        )}
      </Field>
      <Button
        variant="danger"
        disabled={disabled || typed !== expect}
        onClick={() => {
          onConfirm()
          setTyped('')
        }}
      >
        {actionLabel}
      </Button>
    </div>
  )
}

export function DangerZoneTab({ settings }: { settings: SettingsResponse }) {
  return (
    <>
      <RegenerateCard />
      <UsernameCard settings={settings} />
      <ClearDataCard settings={settings} />
      <DeleteAccountCard settings={settings} />
    </>
  )
}

function RegenerateCard() {
  const action = useSettingsAction(settingsApi.regenerateSummaries, { invalidate: [] })

  return (
    <Card title="Regenerate summaries">
      <p className="text-sm text-fg-muted">
        Throws away the pre-computed summaries and rebuilds them from your heartbeats. Do
        this after changing your heartbeat timeout or your aliases. Your raw data is not
        touched.
      </p>
      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button disabled={action.state === 'pending'} onClick={() => action.run(undefined as never)}>
          Regenerate
        </Button>
      </FormActions>
    </Card>
  )
}

function UsernameCard({ settings }: { settings: SettingsResponse }) {
  const [username, setUsername] = useState('')

  const action = useSettingsAction(settingsApi.changeUsername, {
    onSuccess: (result) => {
      // A local session is keyed on the old username, so it cannot survive the change.
      if ((result as { relogin_required: boolean }).relogin_required) {
        window.location.assign(appUrl('/login'))
      }
    },
  })

  return (
    <Card title="Change username">
      <p className="text-sm text-fg-muted">
        Your username is part of every badge and public URL, so old links stop working.
        You will be signed out and have to log in again.
      </p>

      <div className="mt-3 flex flex-wrap items-end gap-3">
        <Field label="New username" hint="No spaces." className="min-w-56 flex-1">
          {(id) => (
            <TextInput
              id={id}
              value={username}
              placeholder={settings.user.id}
              onChange={(event) => setUsername(event.target.value)}
            />
          )}
        </Field>
        <Button
          variant="danger"
          disabled={!username.trim() || username.trim() === settings.user.id || action.state === 'pending'}
          onClick={() => action.run({ username: username.trim() })}
        >
          Change username
        </Button>
      </div>

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        {null}
      </FormActions>
    </Card>
  )
}

function ClearDataCard({ settings }: { settings: SettingsResponse }) {
  const action = useSettingsAction(settingsApi.clearData, { invalidate: [] })

  return (
    <Card title="Delete all coding data">
      <p className="text-sm text-fg-muted">
        Removes every heartbeat, duration and summary. Your account and settings stay.
        This cannot be undone.
      </p>
      <div className="mt-3">
        <ConfirmBox
          expect={settings.user.id}
          label="Type your username to confirm"
          hint="This deletes all of your tracked activity."
          actionLabel="Delete my data"
          disabled={action.state === 'pending'}
          onConfirm={() => action.run(undefined as never)}
        />
      </div>
      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        {null}
      </FormActions>
    </Card>
  )
}

function DeleteAccountCard({ settings }: { settings: SettingsResponse }) {
  const action = useSettingsAction(settingsApi.deleteAccount, {
    invalidate: [],
    onSuccess: () => window.location.assign(appUrl('/')),
  })

  return (
    <Card title="Delete account">
      <p className="text-sm text-fg-muted">
        Removes your account and everything belonging to it. This cannot be undone.
      </p>
      <div className="mt-3">
        <ConfirmBox
          expect={settings.user.id}
          label="Type your username to confirm"
          hint="Your account and all of its data will be deleted."
          actionLabel="Delete my account"
          disabled={action.state === 'pending'}
          onConfirm={() => action.run(undefined as never)}
        />
      </div>
      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        {null}
      </FormActions>
    </Card>
  )
}
