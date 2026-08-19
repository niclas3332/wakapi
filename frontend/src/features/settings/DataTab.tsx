import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, Select, TextInput, Toggle } from '@/components/ui/Form'
import { EmptyState } from '@/components/ui/State'

/**
 * Entity types, matching the constants in models/summary.go. Only the types an alias
 * can actually target are offered.
 */
const ALIAS_TYPES = [
  { value: 0, label: 'Project' },
  { value: 1, label: 'Language' },
  { value: 2, label: 'Editor' },
  { value: 3, label: 'Operating system' },
  { value: 4, label: 'Machine' },
] as const

function aliasTypeLabel(type: number): string {
  return ALIAS_TYPES.find((entry) => entry.value === type)?.label ?? 'Unknown'
}

export function DataTab({ settings }: { settings: SettingsResponse }) {
  return (
    <>
      <UnknownProjectsCard settings={settings} />
      <AliasesCard settings={settings} />
      <LabelsCard settings={settings} />
      <LanguageMappingsCard settings={settings} />
      <TimeoutCard settings={settings} />
    </>
  )
}

function UnknownProjectsCard({ settings }: { settings: SettingsResponse }) {
  const action = useSettingsAction(settingsApi.updateExcludeUnknownProjects)

  return (
    <Card title="Unknown projects">
      <Toggle
        label="Exclude heartbeats without a project"
        hint="Editing a loose file outside any project stops counting towards your totals. Changing this regenerates your summaries."
        checked={settings.user.exclude_unknown_projects}
        disabled={action.state === 'pending'}
        onChange={(enabled) => action.run({ enabled })}
      />
      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        {null}
      </FormActions>
    </Card>
  )
}

function AliasesCard({ settings }: { settings: SettingsResponse }) {
  const [key, setKey] = useState('')
  const [value, setValue] = useState('')
  const [type, setType] = useState<number>(0)

  const create = useSettingsAction(settingsApi.createAlias, {
    onSuccess: () => {
      setKey('')
      setValue('')
    },
  })
  const remove = useSettingsAction((input: { key: string; type: number }) =>
    settingsApi.deleteAlias(input.key, input.type),
  )

  return (
    <Card title="Aliases" meta="merge two names into one">
      <p className="text-sm text-fg-muted">
        Anything tracked under the original name is counted under the alias instead. Use{' '}
        <code className="text-fg">*</code> to match a group, e.g.{' '}
        <code className="text-fg">wakapi-*</code>.
      </p>

      {settings.aliases.length === 0 ? (
        <EmptyState message="No aliases yet." />
      ) : (
        <ul className="mt-3 divide-y divide-ink-700">
          {settings.aliases.map((alias) => (
            <li key={`${alias.type}-${alias.key}`} className="flex items-start gap-3 py-2">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-2">
                  <span className="text-sm font-medium text-fg">{alias.key}</span>
                  <span className="eyebrow">{aliasTypeLabel(alias.type)}</span>
                </div>
                <div className="mt-0.5 truncate text-xs text-fg-subtle">
                  from {alias.values.join(', ')}
                </div>
              </div>
              <Button
                variant="danger"
                disabled={remove.state === 'pending'}
                onClick={() => remove.run({ key: alias.key, type: alias.type })}
              >
                Remove
              </Button>
            </li>
          ))}
        </ul>
      )}

      <div className="mt-4 grid gap-3 sm:grid-cols-[1fr_1fr_10rem]">
        <Field label="Original name" hint="What your editor reports.">
          {(id) => (
            <TextInput id={id} value={value} onChange={(event) => setValue(event.target.value)} />
          )}
        </Field>
        <Field label="Shown as" hint="What you want to see.">
          {(id) => <TextInput id={id} value={key} onChange={(event) => setKey(event.target.value)} />}
        </Field>
        <Field label="Type">
          {(id) => (
            <Select id={id} value={type} onChange={(event) => setType(Number(event.target.value))}>
              {ALIAS_TYPES.map((entry) => (
                <option key={entry.value} value={entry.value}>
                  {entry.label}
                </option>
              ))}
            </Select>
          )}
        </Field>
      </div>

      <FormActions
        status={
          <>
            <ActionStatus state={create.state} message={create.message} />
            <ActionStatus state={remove.state} message={remove.message} />
          </>
        }
      >
        <Button
          variant="primary"
          disabled={!key.trim() || !value.trim() || create.state === 'pending'}
          onClick={() => create.run({ key: key.trim(), value: value.trim(), type })}
        >
          Add alias
        </Button>
      </FormActions>
    </Card>
  )
}

function LabelsCard({ settings }: { settings: SettingsResponse }) {
  const [label, setLabel] = useState('')
  const [selected, setSelected] = useState<string[]>([])

  const create = useSettingsAction(settingsApi.createLabels, {
    onSuccess: () => {
      setLabel('')
      setSelected([])
    },
  })
  const remove = useSettingsAction((input: { label: string; project: string }) =>
    settingsApi.deleteLabel(input.label, input.project),
  )

  return (
    <Card title="Project labels" meta="group projects together">
      <p className="text-sm text-fg-muted">
        A label groups several projects so you can filter on all of them at once.
      </p>

      {settings.labels.length === 0 ? (
        <EmptyState message="No labels yet." />
      ) : (
        <ul className="mt-3 space-y-3">
          {settings.labels.map((group) => (
            <li key={group.label}>
              <div className="text-sm font-medium text-fg">{group.label}</div>
              <ul className="mt-1.5 flex flex-wrap gap-1.5">
                {group.projects.map((project) => (
                  <li key={project}>
                    {/* The chip is the project; its × removes just that one assignment. */}
                    <span className="inline-flex items-center gap-1.5 rounded-full border border-ink-700 bg-ink-900 py-1 pr-1 pl-2.5 text-xs text-fg-muted">
                      {project}
                      <button
                        type="button"
                        aria-label={`Remove ${project} from ${group.label}`}
                        disabled={remove.state === 'pending'}
                        onClick={() => remove.run({ label: group.label, project })}
                        className="flex h-4 w-4 items-center justify-center rounded-full text-fg-subtle transition-colors hover:bg-ink-700 hover:text-danger"
                      >
                        ×
                      </button>
                    </span>
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      )}

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <Field label="Label">
          {(id) => (
            <TextInput
              id={id}
              value={label}
              placeholder="work"
              onChange={(event) => setLabel(event.target.value)}
            />
          )}
        </Field>
        <Field
          label="Projects"
          hint={
            selected.length > 0
              ? `${selected.length} selected - hold ⌘ or Ctrl to pick several.`
              : 'Hold ⌘ or Ctrl to pick several.'
          }
        >
          {(id) => (
            <select
              id={id}
              multiple
              size={Math.min(6, Math.max(3, settings.projects.length))}
              value={selected}
              onChange={(event) =>
                setSelected(Array.from(event.target.selectedOptions, (option) => option.value))
              }
              className="w-full rounded border border-ink-700 bg-ink-900 px-3 py-2 text-sm text-fg"
            >
              {settings.projects.map((project) => (
                <option key={project} value={project}>
                  {project}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>

      <FormActions
        status={
          <>
            <ActionStatus state={create.state} message={create.message} />
            <ActionStatus state={remove.state} message={remove.message} />
          </>
        }
      >
        <Button
          variant="primary"
          disabled={!label.trim() || selected.length === 0 || create.state === 'pending'}
          onClick={() => create.run({ label: label.trim(), projects: selected })}
        >
          Assign label
        </Button>
      </FormActions>
    </Card>
  )
}

function LanguageMappingsCard({ settings }: { settings: SettingsResponse }) {
  const [extension, setExtension] = useState('')
  const [language, setLanguage] = useState('')

  const create = useSettingsAction(settingsApi.createLanguageMapping, {
    onSuccess: () => {
      setExtension('')
      setLanguage('')
    },
  })
  const remove = useSettingsAction(settingsApi.deleteLanguageMapping)

  return (
    <Card title="Language mappings" meta="name a file extension">
      <p className="text-sm text-fg-muted">
        Tell wakapi which language a file extension belongs to when your editor does not.
      </p>

      {settings.language_mappings.length === 0 ? (
        <EmptyState message="No custom mappings. Your editor's own detection is used." />
      ) : (
        <ul className="mt-3 divide-y divide-ink-700">
          {settings.language_mappings.map((mapping) => (
            <li key={mapping.id} className="flex items-center gap-3 py-2 text-sm">
              <code className="text-fg">.{mapping.extension}</code>
              <span className="text-fg-subtle">→</span>
              <span className="text-fg">{mapping.language}</span>
              <Button
                variant="danger"
                className="ml-auto"
                disabled={remove.state === 'pending'}
                onClick={() => remove.run(mapping.id)}
              >
                Remove
              </Button>
            </li>
          ))}
        </ul>
      )}

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <Field label="Extension" hint="With or without the leading dot.">
          {(id) => (
            <TextInput
              id={id}
              value={extension}
              placeholder="vue"
              onChange={(event) => setExtension(event.target.value)}
            />
          )}
        </Field>
        <Field label="Language">
          {(id) => (
            <TextInput
              id={id}
              value={language}
              placeholder="Vue"
              onChange={(event) => setLanguage(event.target.value)}
            />
          )}
        </Field>
      </div>

      <FormActions
        status={
          <>
            <ActionStatus state={create.state} message={create.message} />
            <ActionStatus state={remove.state} message={remove.message} />
          </>
        }
      >
        <Button
          variant="primary"
          disabled={!extension.trim() || !language.trim() || create.state === 'pending'}
          onClick={() => create.run({ extension: extension.trim(), language: language.trim() })}
        >
          Add mapping
        </Button>
      </FormActions>
    </Card>
  )
}

function TimeoutCard({ settings }: { settings: SettingsResponse }) {
  const [minutes, setMinutes] = useState(Math.round(settings.user.heartbeats_timeout_sec / 60))
  const action = useSettingsAction(settingsApi.updateHeartbeatsTimeout)

  return (
    <Card title="Heartbeat timeout">
      <p className="text-sm text-fg-muted">
        How long a pause may be before wakapi treats it as the end of a coding session.
        Longer counts more time; shorter counts less.
      </p>

      <div className="mt-3 max-w-48">
        <Field label="Minutes" hint="Between 1 and 60.">
          {(id) => (
            <TextInput
              id={id}
              type="number"
              min={1}
              max={60}
              value={minutes}
              onChange={(event) => setMinutes(Number(event.target.value))}
            />
          )}
        </Field>
      </div>

      <FormActions status={<ActionStatus state={action.state} message={action.message} />}>
        <Button
          variant="primary"
          disabled={action.state === 'pending'}
          onClick={() => action.run({ minutes })}
        >
          Save timeout
        </Button>
      </FormActions>
    </Card>
  )
}
