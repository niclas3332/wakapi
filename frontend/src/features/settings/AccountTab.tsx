import { useMemo, useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { serverConfig } from '@/lib/bootstrap'
import { formatDate } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, Select, TextInput, Toggle } from '@/components/ui/Form'
import { PasskeysCard } from './PasskeysCard'

const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

/**
 * Every IANA zone the browser knows, which is a better source than a checked-in list:
 * the list ships with the runtime and cannot go stale. Older engines without
 * supportedValuesOf fall back to whatever the account already has.
 */
function timezones(current: string): string[] {
  const supported =
    typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : []
  const all = supported.length > 0 ? supported : [current || 'UTC']
  return all.includes(current) || !current ? all : [current, ...all]
}

export function AccountTab({ settings }: { settings: SettingsResponse }) {
  const user = settings.user
  const isLocal = user.auth_type === 'local'

  return (
    <>
      <ProfileCard settings={settings} />
      {isLocal && <PasswordCard />}
      <PasskeysCard settings={settings} />
      {serverConfig.features.invitesEnabled && <InviteCard />}
      <Card title="Account">
        <dl className="grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt className="eyebrow">Username</dt>
            <dd className="text-fg">{user.id}</dd>
          </div>
          <div>
            <dt className="eyebrow">Signed up</dt>
            <dd className="tabular text-fg">{formatDate(user.created_at)}</dd>
          </div>
          <div>
            <dt className="eyebrow">Sign-in method</dt>
            <dd className="text-fg">{isLocal ? 'Password' : user.auth_type}</dd>
          </div>
        </dl>
      </Card>
    </>
  )
}

function ProfileCard({ settings }: { settings: SettingsResponse }) {
  const user = settings.user
  const [email, setEmail] = useState(user.email)
  const [location, setLocation] = useState(user.location)
  const [startOfWeek, setStartOfWeek] = useState(user.start_of_week)
  const [reportsWeekly, setReportsWeekly] = useState(user.reports_weekly)

  const zones = useMemo(() => timezones(user.location), [user.location])
  const action = useSettingsAction(settingsApi.updateProfile)

  return (
    <Card title="Profile">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label="E-mail address"
          hint={
            user.has_active_subscription
              ? 'Required while your subscription is active.'
              : 'Used for the weekly report and password resets. Optional.'
          }
        >
          {(id) => (
            <TextInput
              id={id}
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
            />
          )}
        </Field>

        <Field label="Time zone" hint="Decides where your days start and end.">
          {(id) => (
            <Select id={id} value={location} onChange={(event) => setLocation(event.target.value)}>
              {zones.map((zone) => (
                <option key={zone} value={zone}>
                  {zone}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <Field label="Week starts on">
          {(id) => (
            <Select
              id={id}
              value={startOfWeek}
              onChange={(event) => setStartOfWeek(Number(event.target.value))}
            >
              {WEEKDAYS.map((day, index) => (
                <option key={day} value={index}>
                  {day}
                </option>
              ))}
            </Select>
          )}
        </Field>
      </div>

      <div className="mt-2">
        <Toggle
          label="Send me a weekly report"
          hint={email ? 'Every Friday evening.' : 'Needs an e-mail address.'}
          checked={reportsWeekly}
          disabled={!email}
          onChange={setReportsWeekly}
        />
      </div>

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button
          variant="primary"
          disabled={action.state === 'pending'}
          onClick={() =>
            action.run({
              email,
              location,
              start_of_week: startOfWeek,
              reports_weekly: reportsWeekly,
              public_leaderboard: settings.user.public_leaderboard,
            })
          }
        >
          Save profile
        </Button>
      </FormActions>
    </Card>
  )
}

function PasswordCard() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [repeat, setRepeat] = useState('')

  const action = useSettingsAction(settingsApi.changePassword, {
    invalidate: [],
    onSuccess: () => {
      setCurrent('')
      setNext('')
      setRepeat('')
    },
  })

  const mismatch = repeat.length > 0 && next !== repeat

  return (
    <Card title="Password">
      <div className="grid gap-4 sm:grid-cols-3">
        <Field label="Current password">
          {(id) => (
            <TextInput
              id={id}
              type="password"
              autoComplete="current-password"
              value={current}
              onChange={(event) => setCurrent(event.target.value)}
            />
          )}
        </Field>
        <Field label="New password" hint="At least 6 characters.">
          {(id) => (
            <TextInput
              id={id}
              type="password"
              autoComplete="new-password"
              value={next}
              onChange={(event) => setNext(event.target.value)}
            />
          )}
        </Field>
        <Field label="Repeat new password" hint={mismatch ? 'The two entries differ.' : undefined}>
          {(id) => (
            <TextInput
              id={id}
              type="password"
              autoComplete="new-password"
              value={repeat}
              onChange={(event) => setRepeat(event.target.value)}
            />
          )}
        </Field>
      </div>

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button
          variant="primary"
          disabled={action.state === 'pending' || !current || !next || mismatch}
          onClick={() =>
            action.run({ password_old: current, password_new: next, password_repeat: repeat })
          }
        >
          Change password
        </Button>
      </FormActions>
    </Card>
  )
}

function InviteCard() {
  const [link, setLink] = useState('')
  const [copied, setCopied] = useState(false)

  const action = useSettingsAction(
    async () => {
      const result = await settingsApi.createInvite()
      return { ...result, message: 'Invite link created.' }
    },
    { invalidate: [], onSuccess: (result) => setLink((result as { link: string }).link) },
  )

  return (
    <Card title="Invite a friend">
      <p className="text-sm text-fg-muted">
        An invite link lets one person sign up on this instance, even when open signup is
        off.
      </p>

      {link && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <code className="min-w-0 flex-1 truncate rounded border border-ink-700 bg-ink-900 px-3 py-2 text-xs text-fg">
            {link}
          </code>
          <Button
            onClick={() => {
              navigator.clipboard.writeText(link).then(
                () => setCopied(true),
                () => setCopied(false),
              )
            }}
          >
            {copied ? 'Copied' : 'Copy'}
          </Button>
        </div>
      )}

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button disabled={action.state === 'pending'} onClick={() => action.run(undefined as never)}>
          Create invite link
        </Button>
      </FormActions>
    </Card>
  )
}
