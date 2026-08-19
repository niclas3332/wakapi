import { useEffect, useState } from 'react'

import { settingsApi, useBranding, useSettingsAction } from '@/api/settings'
import type { Branding } from '@/api/types'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, TextInput } from '@/components/ui/Form'
import { ErrorState, LoadingRows } from '@/components/ui/State'

type FieldSpec = {
  key: keyof Branding
  label: string
  hint?: string
  type?: 'text' | 'url' | 'color'
}

const IDENTITY: FieldSpec[] = [
  { key: 'instance_name', label: 'Instance name', hint: 'Shown next to the logo.' },
  {
    key: 'logo_url',
    label: 'Logo',
    hint: 'An https URL, an absolute path, or a data:image/svg+xml URI.',
    type: 'url',
  },
  { key: 'footer_text', label: 'Footer text' },
  { key: 'footer_url', label: 'Footer link', type: 'url' },
]

const BADGES: FieldSpec[] = [
  {
    key: 'badge_label',
    label: 'Badge label',
    hint: 'The left half of /api/badge/... Overridable per request with ?label=.',
  },
  {
    key: 'badge_color',
    label: 'Badge color',
    hint: 'A hex triplet or a shields.io colour name.',
  },
  { key: 'readme_card_title', label: 'Readme card title' },
]

const ACTIVITY: FieldSpec[] = [
  { key: 'activity_title', label: 'Attribution title' },
  { key: 'activity_logo_url', label: 'Attribution logo', hint: 'Empty leaves it off.', type: 'url' },
  { key: 'activity_color_min_dark', label: 'Dark: no activity', type: 'color' },
  { key: 'activity_color_max_dark', label: 'Dark: busiest day', type: 'color' },
  { key: 'activity_color_min_light', label: 'Light: no activity', type: 'color' },
  { key: 'activity_color_max_light', label: 'Light: busiest day', type: 'color' },
]

export function AdminTab() {
  const query = useBranding(true)
  const [draft, setDraft] = useState<Branding | null>(null)
  const [rejected, setRejected] = useState<string[]>([])

  // The form holds the stored overrides, not the effective values, and shows what is
  // in effect as the placeholder. Blank therefore keeps meaning "derive this", so
  // renaming the instance later renames everything that follows from it.
  useEffect(() => {
    if (query.data && !draft) setDraft(query.data.overrides)
  }, [query.data, draft])

  const action = useSettingsAction(settingsApi.updateBranding, {
    invalidate: [['branding']],
    onSuccess: (result) => setRejected((result as { rejected: string[] }).rejected ?? []),
  })

  if (query.isError) {
    return (
      <ErrorState
        message={`Could not load the branding: ${(query.error as Error).message}`}
        onRetry={() => query.refetch()}
      />
    )
  }
  if (query.isPending || !draft || !query.data) {
    return (
      <Card>
        <LoadingRows rows={6} />
      </Card>
    )
  }

  const effective = query.data.effective
  const update = (key: keyof Branding, value: string) =>
    setDraft((current) => (current ? { ...current, [key]: value } : current))

  function renderFields(specs: FieldSpec[]) {
    return (
      <div className="grid gap-4 sm:grid-cols-2">
        {specs.map((spec) => {
          const inherited = draft![spec.key] === ''
          return (
            <Field
              key={spec.key}
              label={spec.label}
              hint={
                rejected.includes(spec.key)
                  ? 'Rejected - the default is in use.'
                  : inherited
                    ? `In use: ${effective[spec.key] || 'none'}`
                    : spec.hint
              }
            >
              {(id) =>
                spec.type === 'color' ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      aria-label={spec.label}
                      value={draft![spec.key] || effective[spec.key]}
                      onChange={(event) => update(spec.key, event.target.value)}
                      className="h-9 w-10 shrink-0 rounded border border-ink-700 bg-ink-900"
                    />
                    <TextInput
                      id={id}
                      value={draft![spec.key]}
                      placeholder={effective[spec.key]}
                      onChange={(event) => update(spec.key, event.target.value)}
                    />
                  </div>
                ) : (
                  <TextInput
                    id={id}
                    type={spec.type === 'url' ? 'url' : 'text'}
                    value={draft![spec.key]}
                    placeholder={effective[spec.key]}
                    onChange={(event) => update(spec.key, event.target.value)}
                  />
                )
              }
            </Field>
          )
        })}
      </div>
    )
  }

  const save = (
    <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
      <Button
        onClick={() => {
          // Clearing every field is what "use the config file" means here: blank
          // fields inherit, so this drops the overrides rather than pinning them.
          setDraft(Object.fromEntries(Object.keys(draft!).map((key) => [key, ''])) as Branding)
          setRejected([])
        }}
      >
        Clear all overrides
      </Button>
      <Button
        variant="primary"
        disabled={action.state === 'pending'}
        onClick={() => action.run(draft!)}
      >
        Save branding
      </Button>
    </FormActions>
  )

  return (
    <>
      <Card title="Instance identity">
        <p className="text-sm text-fg-muted">
          These apply to everyone on this instance. A field left blank falls back to the
          config file, and the badge label, readme card title and chart attribution fall
          back to the instance name — so renaming the instance renames all three. Reload
          the page after saving to see the header change.
        </p>
        <div className="mt-4">{renderFields(IDENTITY)}</div>
        <div className="mt-4 flex items-center gap-2 rounded border border-ink-700 bg-ink-900 px-3 py-2">
          {draft.logo_url || effective.logo_url ? (
            <img src={draft.logo_url || effective.logo_url} alt="" className="h-5 w-5 object-contain" />
          ) : (
            <span
              aria-hidden="true"
              className="inline-block h-5 w-5 rounded-full border-2 border-accent border-r-transparent"
            />
          )}
          <span className="text-sm text-fg">{draft.instance_name || effective.instance_name}</span>
          <span className="eyebrow ml-auto">preview</span>
        </div>
      </Card>

      <Card title="Badges and cards">{renderFields(BADGES)}</Card>

      <Card title="Activity chart">
        <p className="text-sm text-fg-muted">
          Days are shaded between these two colours by quartile, so the busiest quarter of
          your active days gets the strongest one.
        </p>
        <div className="mt-4">{renderFields(ACTIVITY)}</div>
        {save}
      </Card>
    </>
  )
}
