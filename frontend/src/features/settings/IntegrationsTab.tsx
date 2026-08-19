import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { serverConfig } from '@/lib/bootstrap'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, TextInput, Toggle } from '@/components/ui/Form'

const README_STATS_DEFAULT = 'https://github-readme-stats.vercel.app'

export function IntegrationsTab({ settings }: { settings: SettingsResponse }) {
  return (
    <>
      <WakatimeCard settings={settings} />
      <BadgesCard settings={settings} />
      <ReadmeStatsCard settings={settings} />
    </>
  )
}

function WakatimeCard({ settings }: { settings: SettingsResponse }) {
  const connected = settings.user.wakatime_connected
  const [apiKey, setApiKey] = useState('')
  const [apiUrl, setApiUrl] = useState(settings.user.wakatime_api_url)
  const [legacyImporter, setLegacyImporter] = useState(false)

  const connect = useSettingsAction(settingsApi.updateWakatime, {
    onSuccess: () => setApiKey(''),
  })
  const runImport = useSettingsAction(settingsApi.importWakatime, { invalidate: [] })

  return (
    <Card title="WakaTime" meta={connected ? 'connected' : 'not connected'}>
      <p className="text-sm text-fg-muted">
        Relay every heartbeat this instance receives on to WakaTime, so you can use both
        at once. Point the URL at another wakapi instance's{' '}
        <code className="text-fg">/api/compat/wakatime/v1</code> to forward there instead.
      </p>
      <p className="mt-2 text-xs text-fg-subtle">
        The operators of this server can read your WakaTime data while this is on. Leave it
        off if that is not acceptable to you.
      </p>

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <Field label="API URL" hint={`Leave empty for ${settings.wakatime_api_url}.`}>
          {(id) => (
            <TextInput
              id={id}
              type="url"
              value={apiUrl}
              disabled={connected}
              placeholder={settings.wakatime_api_url}
              onChange={(event) => setApiUrl(event.target.value)}
            />
          )}
        </Field>
        <Field
          label="API key"
          hint={connected ? 'Stored. Disconnect to replace it.' : 'From your WakaTime account settings.'}
        >
          {(id) => (
            <TextInput
              id={id}
              type="password"
              autoComplete="off"
              value={apiKey}
              disabled={connected}
              placeholder={connected ? '••••••••' : 'waka_...'}
              onChange={(event) => setApiKey(event.target.value)}
            />
          )}
        </Field>
      </div>

      {connected && serverConfig.features.importEnabled && (
        <div className="mt-3">
          <Toggle
            label="Use the legacy importer"
            hint="Slower, but works when the normal import keeps failing."
            checked={legacyImporter}
            onChange={setLegacyImporter}
          />
        </div>
      )}

      <FormActions
        status={
          <>
            <ActionStatus state={connect.state} message={connect.message} />
            <ActionStatus state={runImport.state} message={runImport.message} />
          </>
        }
      >
        {connected ? (
          <>
            {serverConfig.features.importEnabled && (
              <Button
                disabled={runImport.state === 'pending'}
                onClick={() => runImport.run({ use_legacy_importer: legacyImporter })}
              >
                Import my WakaTime data
              </Button>
            )}
            <Button
              variant="danger"
              disabled={connect.state === 'pending'}
              onClick={() => connect.run({ api_key: '', api_url: '' })}
            >
              Disconnect
            </Button>
          </>
        ) : (
          <Button
            variant="primary"
            disabled={!apiKey.trim() || connect.state === 'pending'}
            onClick={() => connect.run({ api_key: apiKey.trim(), api_url: apiUrl.trim() })}
          >
            Connect
          </Button>
        )}
      </FormActions>
    </Card>
  )
}

/** A badge url with its live preview and a copy button. */
function BadgeRow({ url, alt }: { url: string; alt: string }) {
  const [copied, setCopied] = useState(false)

  return (
    <li className="flex flex-wrap items-center gap-3 py-2">
      <img src={url} alt={alt} className="h-5 shrink-0" />
      <code className="min-w-0 flex-1 truncate rounded border border-ink-700 bg-ink-900 px-3 py-1.5 text-xs text-fg-muted">
        {url}
      </code>
      <Button
        onClick={() =>
          navigator.clipboard.writeText(url).then(
            () => setCopied(true),
            () => setCopied(false),
          )
        }
      >
        {copied ? 'Copied' : 'Copy'}
      </Button>
    </li>
  )
}

function BadgesCard({ settings }: { settings: SettingsResponse }) {
  const user = settings.user
  const base = serverConfig.publicUrl.replace(/\/$/, '')
  const shares = user.sharing.max_days !== 0

  const badges = [
    { url: `${base}/api/badge/${user.id}/interval:today?label=today`, alt: 'Badge for today' },
    {
      url: `${base}/api/badge/${user.id}/interval:30_days?label=last%2030d`,
      alt: 'Badge for the last 30 days',
    },
    {
      url: `https://img.shields.io/endpoint?url=${encodeURIComponent(`${base}/api/compat/shields/v1/${user.id}/interval:all_time`)}&label=All%20time&color=blue`,
      alt: 'Shields.io badge for all time',
    },
  ]

  return (
    <Card title="Badges" meta="for a readme or a forum signature">
      <p className="text-sm text-fg-muted">
        Adapt the <code className="text-fg">label</code> and{' '}
        <code className="text-fg">color</code> parameters to taste. The shields.io endpoint
        adds their styles on top, and needs an instance the internet can reach.
      </p>

      {shares ? (
        <ul className="mt-3 divide-y divide-ink-700">
          {badges.map((badge) => (
            <BadgeRow key={badge.url} url={badge.url} alt={badge.alt} />
          ))}
        </ul>
      ) : (
        <p className="mt-3 rounded border border-ink-700 bg-ink-900 px-3 py-2 text-sm text-fg-muted">
          Badges need public access to your data. Set a visible range under Permissions
          first.
        </p>
      )}
    </Card>
  )
}

function ReadmeStatsCard({ settings }: { settings: SettingsResponse }) {
  const user = settings.user
  const [baseUrl, setBaseUrl] = useState(user.readme_stats_base_url)
  const action = useSettingsAction(settingsApi.updateReadmeStatsBaseUrl)

  const host = serverConfig.publicUrl.replace(/^https?:\/\//, '').replace(/\/$/, '')
  const cardUrl =
    `${user.readme_stats_base_url || README_STATS_DEFAULT}/api/wakatime` +
    `?username=${encodeURIComponent(user.id)}&api_domain=${encodeURIComponent(host)}` +
    `&bg_color=1A202C&title_color=2F855A&icon_color=2F855A&text_color=ffffff` +
    `&custom_title=${encodeURIComponent(settings.readme_card_title)}&layout=compact`

  return (
    <Card title="GitHub readme stats" meta="a summary card for your profile">
      <p className="text-sm text-fg-muted">
        Which card you get depends on how much you share: 7 days or more gives a weekly
        card, 366 days a yearly one, everything an all-time one. Needs an instance the
        internet can reach.
      </p>

      {user.sharing.max_days === 0 ? (
        <p className="mt-3 rounded border border-ink-700 bg-ink-900 px-3 py-2 text-sm text-fg-muted">
          This card needs public access to your data. Set a visible range under Permissions
          first.
        </p>
      ) : (
        <>
          <img src={cardUrl} alt="Readme stats card preview" className="mt-3 max-w-full rounded" />
          <textarea
            readOnly
            rows={4}
            value={cardUrl}
            className="mt-3 w-full resize-none rounded border border-ink-700 bg-ink-900 px-3 py-2 text-xs text-fg-muted"
          />
        </>
      )}

      <div className="mt-4 max-w-md">
        <Field label="Custom github-readme-stats base URL" hint="Empty uses the public instance.">
          {(id) => (
            <TextInput
              id={id}
              type="url"
              value={baseUrl}
              placeholder={README_STATS_DEFAULT}
              onChange={(event) => setBaseUrl(event.target.value)}
            />
          )}
        </Field>
      </div>

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button
          variant="primary"
          disabled={action.state === 'pending'}
          onClick={() => action.run({ base_url: baseUrl.trim() })}
        >
          Save
        </Button>
      </FormActions>
    </Card>
  )
}
