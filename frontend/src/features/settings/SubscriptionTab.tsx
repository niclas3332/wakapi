import type { SettingsResponse } from '@/api/types'
import { appUrl } from '@/api/client'
import { serverConfig } from '@/lib/bootstrap'
import { formatDate } from '@/lib/format'
import { Card } from '@/components/ui/Card'

/**
 * Checkout stays server-rendered.
 *
 * These are real form posts to the existing Stripe handlers rather than fetch calls:
 * the server answers with a redirect to Stripe, and a browser can only follow that
 * from a navigation. Routing money through a JSON layer would buy nothing and add a
 * place for it to go wrong.
 */
export function SubscriptionTab({ settings }: { settings: SettingsResponse }) {
  const user = settings.user
  const retention = serverConfig.dataRetentionMonths

  return (
    <>
      <Card title="Subscription">
        <p className="text-sm text-fg-muted">
          Wakapi is free to use. A subscription for {settings.subscription_price} per month
          keeps your full history instead of the standard retention window, and you can
          cancel at any time.
        </p>

        {retention > 0 && (
          <p className="mt-3 rounded border border-ink-700 bg-ink-900 px-3 py-2 text-sm text-fg-muted">
            Without a subscription, coding activity older than {retention} months is
            deleted daily. With one, everything is kept.
          </p>
        )}

        <div className="mt-4">
          {user.has_active_subscription ? (
            <div className="space-y-3">
              <p className="text-sm text-fg">
                Active
                {user.subscribed_until && ` until ${formatDate(user.subscribed_until)}`}.
              </p>
              <form method="post" action={appUrl('/subscription/portal')}>
                <button
                  type="submit"
                  className="rounded bg-accent-strong px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent"
                >
                  Manage subscription
                </button>
              </form>
            </div>
          ) : user.email ? (
            <form method="post" action={appUrl('/subscription/checkout')}>
              <button
                type="submit"
                className="rounded bg-accent-strong px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent"
              >
                Subscribe for {settings.subscription_price} / month
              </button>
            </form>
          ) : (
            <p className="text-sm text-fg-muted">
              Add an e-mail address under Account before subscribing.
            </p>
          )}
        </div>

        <p className="mt-4 text-xs text-fg-subtle">
          Your e-mail address and any billing details you enter are processed by Stripe
          under their{' '}
          <a
            href="https://stripe.com/privacy"
            target="_blank"
            rel="noreferrer noopener"
            className="text-accent hover:text-fg"
          >
            privacy policy
          </a>
          . A new subscription can take a minute to show up here.
          {settings.support_contact && (
            <>
              {' '}
              Problems?{' '}
              <a href={`mailto:${settings.support_contact}`} className="text-accent hover:text-fg">
                {settings.support_contact}
              </a>
            </>
          )}
        </p>
      </Card>
    </>
  )
}
