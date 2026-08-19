import { useSearchParams } from 'react-router'
import clsx from 'clsx'

import { useSettings } from '@/api/settings'
import { serverConfig } from '@/lib/bootstrap'
import { ErrorState, LoadingRows } from '@/components/ui/State'
import { Card } from '@/components/ui/Card'
import { AccountTab } from '@/features/settings/AccountTab'
import { DataTab } from '@/features/settings/DataTab'
import { PermissionsTab } from '@/features/settings/PermissionsTab'
import { IntegrationsTab } from '@/features/settings/IntegrationsTab'
import { ApiKeysTab } from '@/features/settings/ApiKeysTab'
import { SubscriptionTab } from '@/features/settings/SubscriptionTab'
import { DangerZoneTab } from '@/features/settings/DangerZoneTab'
import { AdminTab } from '@/features/settings/AdminTab'

type TabId = 'account' | 'data' | 'permissions' | 'integrations' | 'api-keys' | 'subscription' | 'admin' | 'danger'

const TABS: { id: TabId; label: string }[] = [
  { id: 'account', label: 'Account' },
  { id: 'data', label: 'Data' },
  { id: 'permissions', label: 'Permissions' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'api-keys', label: 'API keys' },
  { id: 'subscription', label: 'Subscription' },
  { id: 'admin', label: 'Admin' },
  { id: 'danger', label: 'Danger zone' },
]

export function SettingsPage() {
  const [search, setSearch] = useSearchParams()
  const settingsQuery = useSettings()

  const data = settingsQuery.data
  const isAdmin = data?.user.is_admin ?? false

  const tabs = TABS.filter((tab) => {
    if (tab.id === 'subscription') return serverConfig.features.subscriptionsEnabled
    if (tab.id === 'admin') return isAdmin
    return true
  })

  const requested = search.get('tab') as TabId | null
  const active: TabId = tabs.some((tab) => tab.id === requested) ? requested! : 'account'

  function selectTab(id: TabId) {
    const next = new URLSearchParams(search)
    next.set('tab', id)
    setSearch(next, { replace: true })
  }

  if (settingsQuery.isError) {
    return (
      <ErrorState
        message={`Could not load your settings: ${(settingsQuery.error as Error).message}`}
        onRetry={() => settingsQuery.refetch()}
      />
    )
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-fg">Settings</h1>

      {/*
        Tabs scroll horizontally rather than wrapping: a wrapped tab bar changes height
        as the window narrows and pushes the content around while reading.
      */}
      <nav className="-mx-1 overflow-x-auto">
        <ul className="flex min-w-max gap-1 border-b border-ink-700 px-1">
          {tabs.map((tab) => (
            <li key={tab.id}>
              <button
                type="button"
                onClick={() => selectTab(tab.id)}
                aria-current={active === tab.id ? 'page' : undefined}
                className={clsx(
                  'block border-b-2 px-3 py-2 text-sm transition-colors',
                  active === tab.id
                    ? 'border-accent text-fg'
                    : 'border-transparent text-fg-muted hover:text-fg',
                  tab.id === 'danger' && active !== tab.id && 'text-danger/70 hover:text-danger',
                )}
              >
                {tab.label}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      {settingsQuery.isPending || !data ? (
        <Card>
          <LoadingRows rows={8} />
        </Card>
      ) : (
        <div className="space-y-4">
          {active === 'account' && <AccountTab settings={data} />}
          {active === 'data' && <DataTab settings={data} />}
          {active === 'permissions' && <PermissionsTab settings={data} />}
          {active === 'integrations' && <IntegrationsTab settings={data} />}
          {active === 'api-keys' && <ApiKeysTab settings={data} />}
          {active === 'subscription' && <SubscriptionTab settings={data} />}
          {active === 'admin' && <AdminTab />}
          {active === 'danger' && <DangerZoneTab settings={data} />}
        </div>
      )}
    </div>
  )
}
