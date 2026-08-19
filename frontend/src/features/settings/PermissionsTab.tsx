import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse, Sharing } from '@/api/types'
import { serverConfig } from '@/lib/bootstrap'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, Select, Toggle } from '@/components/ui/Form'

/**
 * How far back a public visitor may look. -1 is "everything" and 0 is "nothing", which
 * is why the control is a list of named windows rather than a number field.
 */
const RANGES = [
  { value: 0, label: 'Nothing' },
  { value: 1, label: 'Last 24 hours' },
  { value: 7, label: 'Last 7 days' },
  { value: 14, label: 'Last 14 days' },
  { value: 30, label: 'Last 30 days' },
  { value: 183, label: 'Last 6 months' },
  { value: 365, label: 'Last 12 months' },
  { value: -1, label: 'Everything' },
]

const CATEGORIES: { key: keyof Omit<Sharing, 'max_days'>; label: string }[] = [
  { key: 'projects', label: 'Projects' },
  { key: 'languages', label: 'Languages' },
  { key: 'editors', label: 'Editors' },
  { key: 'operating_systems', label: 'Operating systems' },
  { key: 'machines', label: 'Machines' },
  { key: 'labels', label: 'Labels' },
  { key: 'activity_chart', label: 'Activity chart' },
]

export function PermissionsTab({ settings }: { settings: SettingsResponse }) {
  return (
    <>
      {serverConfig.features.leaderboardEnabled && <LeaderboardCard settings={settings} />}
      <SharingCard settings={settings} />
    </>
  )
}

function LeaderboardCard({ settings }: { settings: SettingsResponse }) {
  const action = useSettingsAction(settingsApi.updateLeaderboardOptIn)

  return (
    <Card title="Public leaderboard">
      <Toggle
        label="Show me on the leaderboard"
        hint="Your username and total coding time become visible to everyone who can see the leaderboard."
        checked={settings.user.public_leaderboard}
        disabled={action.state === 'pending'}
        onChange={(enabled) => action.run({ enabled })}
      />
      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        {null}
      </FormActions>
    </Card>
  )
}

function SharingCard({ settings }: { settings: SettingsResponse }) {
  const [sharing, setSharing] = useState<Sharing>(settings.user.sharing)
  const action = useSettingsAction(settingsApi.updateSharing)

  const nothingShared = sharing.max_days === 0
  const update = (patch: Partial<Sharing>) => setSharing((current) => ({ ...current, ...patch }))

  return (
    <Card title="Public data" meta="what badges and the public api may show">
      <p className="text-sm text-fg-muted">
        These control what someone who is not signed in can read - through badges, the
        shields.io endpoint and the activity chart image.
      </p>

      <div className="mt-4 max-w-64">
        <Field label="Visible range" hint="Applies to every category below.">
          {(id) => (
            <Select
              id={id}
              value={sharing.max_days}
              onChange={(event) => update({ max_days: Number(event.target.value) })}
            >
              {RANGES.map((range) => (
                <option key={range.value} value={range.value}>
                  {range.label}
                </option>
              ))}
            </Select>
          )}
        </Field>
      </div>

      <div className="mt-3 grid gap-1 sm:grid-cols-2">
        {CATEGORIES.map((category) => (
          <Toggle
            key={category.key}
            label={category.label}
            checked={sharing[category.key]}
            // With nothing shared, the individual switches cannot take effect, so they
            // are disabled rather than left looking meaningful.
            disabled={nothingShared}
            onChange={(value) => update({ [category.key]: value } as Partial<Sharing>)}
          />
        ))}
      </div>

      {nothingShared && (
        <p className="mt-2 text-xs text-fg-subtle">
          Nothing is shared while the range is set to “Nothing”.
        </p>
      )}

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button
          variant="primary"
          disabled={action.state === 'pending'}
          onClick={() => action.run(sharing)}
        >
          Save sharing
        </Button>
      </FormActions>
    </Card>
  )
}
