import { useState } from 'react'

import { settingsApi, useSettingsAction } from '@/api/settings'
import type { SettingsResponse } from '@/api/types'
import { serverConfig } from '@/lib/bootstrap'
import { formatDate } from '@/lib/format'
import { Card } from '@/components/ui/Card'
import { ActionStatus, Button, Field, FormActions, TextInput } from '@/components/ui/Form'
import { EmptyState } from '@/components/ui/State'

/** The subset of the server's registration options the browser API needs. */
type RegistrationOptions = { publicKey: PublicKeyCredentialCreationOptions }

/**
 * base64url is what WebAuthn speaks; the browser hands back ArrayBuffers, and the
 * server expects them re-encoded the same way it sent them.
 */
function fromBase64Url(value: string): ArrayBuffer {
  const padded = value.replace(/-/g, '+').replace(/_/g, '/')
  const binary = atob(padded.padEnd(padded.length + ((4 - (padded.length % 4)) % 4), '='))
  // Allocated rather than built with Uint8Array.from, so the result is backed by a
  // plain ArrayBuffer - which is what the credential API accepts.
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

function toBase64Url(buffer: ArrayBuffer): string {
  const binary = String.fromCharCode(...new Uint8Array(buffer))
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/**
 * Run the browser's registration ceremony and hand the result back as the JSON shape
 * the server parses (protocol.ParseCredentialCreationResponseBytes).
 */
async function register(raw: unknown): Promise<string> {
  const options = (raw as RegistrationOptions).publicKey as unknown as Record<string, unknown>

  const publicKey: PublicKeyCredentialCreationOptions = {
    ...(options as unknown as PublicKeyCredentialCreationOptions),
    challenge: fromBase64Url(options.challenge as string),
    user: {
      ...(options.user as PublicKeyCredentialUserEntity),
      id: fromBase64Url((options.user as { id: string }).id),
    },
    excludeCredentials: ((options.excludeCredentials ?? []) as { id: string; type: string }[]).map(
      (credential) => ({
        ...credential,
        id: fromBase64Url(credential.id),
        type: 'public-key' as const,
      }),
    ),
  }

  const credential = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential | null
  if (!credential) throw new Error('Your browser did not return a passkey.')

  const response = credential.response as AuthenticatorAttestationResponse
  return JSON.stringify({
    id: credential.id,
    rawId: toBase64Url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: toBase64Url(response.attestationObject),
      clientDataJSON: toBase64Url(response.clientDataJSON),
    },
  })
}

export function PasskeysCard({ settings }: { settings: SettingsResponse }) {
  const [name, setName] = useState('')

  const isLocal = settings.user.auth_type === 'local'
  const supported = typeof window !== 'undefined' && !!window.PublicKeyCredential

  const add = useSettingsAction(async (credentialName: string) => {
    const options = await settingsApi.webauthnRegistrationOptions()
    const credential = await register(options)
    return settingsApi.createWebauthnCredential({ name: credentialName, credential })
  })

  const remove = useSettingsAction(settingsApi.deleteWebauthnCredential)

  if (!serverConfig.features.webauthnEnabled) return null

  return (
    <Card title="Passkeys" meta="sign in without a password">
      {settings.webauthn_credentials.length === 0 ? (
        <EmptyState message="No passkeys yet. Add one to sign in with your fingerprint, face or security key." />
      ) : (
        <ul className="divide-y divide-ink-700">
          {settings.webauthn_credentials.map((credential) => (
            <li key={credential.name} className="flex items-center gap-3 py-2">
              <div className="min-w-0">
                <div className="truncate text-sm text-fg">{credential.name}</div>
                <div className="tabular text-xs text-fg-subtle">
                  added {formatDate(credential.created_at)}
                  {credential.last_used_at && ` · last used ${formatDate(credential.last_used_at)}`}
                </div>
              </div>
              <Button
                variant="danger"
                className="ml-auto"
                disabled={remove.state === 'pending'}
                onClick={() => remove.run(credential.name)}
              >
                Remove
              </Button>
            </li>
          ))}
        </ul>
      )}

      {!isLocal && (
        <p className="mt-3 text-xs text-fg-subtle">
          Passkeys are only available for accounts that sign in with a password on this
          instance.
        </p>
      )}
      {isLocal && !supported && (
        <p className="mt-3 text-xs text-fg-subtle">This browser does not support passkeys.</p>
      )}

      {isLocal && supported && (
        <div className="mt-4 flex flex-wrap items-end gap-3">
          <Field label="Name" hint="So you can tell your keys apart later." className="min-w-48 flex-1">
            {(id) => (
              <TextInput
                id={id}
                value={name}
                placeholder="MacBook Touch ID"
                onChange={(event) => setName(event.target.value)}
              />
            )}
          </Field>
          <Button
            variant="primary"
            disabled={!name.trim() || add.state === 'pending'}
            onClick={() => add.run(name.trim())}
          >
            Add passkey
          </Button>
        </div>
      )}

      <FormActions
        status={
          <>
            <ActionStatus state={add.state} message={add.message} />
            <ActionStatus state={remove.state} message={remove.message} />
          </>
        }
      >
        {null}
      </FormActions>
    </Card>
  )
}
