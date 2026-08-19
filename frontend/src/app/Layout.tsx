import { useState } from 'react'
import { NavLink, Outlet } from 'react-router'
import clsx from 'clsx'

import { useBootstrap } from '@/api/queries'
import { appUrl } from '@/api/client'
import { serverConfig } from '@/lib/bootstrap'
import { ApiKeyButton } from '@/components/ui/ApiKeyButton'

type NavItem = { to: string; label: string; enabled?: boolean }

const RESOURCES = [
  { href: serverConfig.branding.footerUrl || 'https://github.com/muety/wakapi', label: 'GitHub' },
  { href: 'https://wakatime.com', label: 'WakaTime' },
]

export function Layout() {
  const { data } = useBootstrap()
  const user = data?.user
  const [menuOpen, setMenuOpen] = useState(false)

  const nav: NavItem[] = [
    { to: '/summary', label: 'Dashboard' },
    { to: '/projects', label: 'Projects' },
    { to: '/leaderboard', label: 'Leaderboard', enabled: serverConfig.features.leaderboardEnabled },
  ].filter((item) => item.enabled !== false)

  return (
    // The sidebar column carries the background, not the sticky aside itself -
    // otherwise the tint stops one screen down on a long page.
    <div className="min-h-screen lg:grid lg:grid-cols-[15rem_1fr] lg:bg-ink-900">
      {/* Sidebar: a rail on large screens, a drawer below. */}
      {/*
        The border lives on the main column, not on the sidebar: the sidebar is
        h-screen so its own border would stop halfway down a long page.
      */}
      <aside
        className={clsx('border-ink-700 bg-ink-900 lg:sticky lg:top-0 lg:h-screen', menuOpen ? 'border-b' : 'border-b-0')}
      >
        <div className="flex items-center gap-2 px-4 py-4 lg:px-5">
          <Brand />
          <button
            type="button"
            className="ml-auto rounded border border-ink-700 px-2 py-1 text-xs text-fg-muted lg:hidden"
            onClick={() => setMenuOpen((value) => !value)}
            aria-expanded={menuOpen}
          >
            Menu
          </button>
        </div>

        <nav className={clsx('px-2 pb-4 lg:block lg:px-3', menuOpen ? 'block' : 'hidden')}>
          <ul className="space-y-0.5">
            {nav.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  onClick={() => setMenuOpen(false)}
                  className={({ isActive }) =>
                    clsx(
                      'block rounded px-3 py-2 text-sm transition-colors',
                      isActive
                        ? 'bg-accent-soft text-fg shadow-[inset_2px_0_0_0_var(--color-accent)]'
                        : 'text-fg-muted hover:bg-ink-800 hover:text-fg',
                    )
                  }
                >
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>

          <div className="mt-6 border-t border-ink-700 pt-4">
            <p className="eyebrow px-3 pb-2">Settings</p>
            <ul className="space-y-0.5">
              {/*
                Settings, setup and the api docs are still server-rendered pages, so
                these are real navigations rather than client-side routes.
              */}
              <li>
                <a
                  href={appUrl('/settings')}
                  className="block rounded px-3 py-2 text-sm text-fg-muted transition-colors hover:bg-ink-800 hover:text-fg"
                >
                  All settings
                </a>
              </li>
              <li>
                <a
                  href={appUrl('/setup')}
                  className="block rounded px-3 py-2 text-sm text-fg-muted transition-colors hover:bg-ink-800 hover:text-fg"
                >
                  Connect an editor
                </a>
              </li>
              <li>
                <a
                  href={appUrl('/swagger-ui/')}
                  className="block rounded px-3 py-2 text-sm text-fg-muted transition-colors hover:bg-ink-800 hover:text-fg"
                >
                  API docs
                </a>
              </li>
            </ul>
          </div>

          <div className="mt-6 border-t border-ink-700 pt-4">
            <ul className="space-y-0.5">
              {RESOURCES.map((item) => (
                <li key={item.href}>
                  <a
                    href={item.href}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="block rounded px-3 py-2 text-xs text-fg-subtle transition-colors hover:text-fg-muted"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </nav>
      </aside>

      <div className="min-w-0 bg-ink-950 lg:border-l lg:border-ink-700">
        <header className="flex items-center gap-3 border-b border-ink-700 px-4 py-3 lg:px-8">
          <div className="ml-auto flex items-center gap-2">
            {user && <ApiKeyButton apiKey={user.api_key} />}
            {user && (
              <span className="hidden text-xs text-fg-muted sm:block" title={user.email}>
                {user.id}
              </span>
            )}
            <form method="post" action={appUrl('/logout')}>
              <button
                type="submit"
                className="rounded border border-ink-700 px-2.5 py-1.5 text-xs text-fg-muted transition-colors hover:border-ink-600 hover:text-fg"
              >
                Log out
              </button>
            </form>
          </div>
        </header>

        <main className="mx-auto max-w-[100rem] px-4 py-6 lg:px-8">
          <Outlet />
        </main>

        {/*
          The footer states what this instance is, not what it runs on: version and
          database are operator facts that belong in logs, not under every page.
        */}
        <footer className="mx-auto max-w-[100rem] px-4 pb-8 text-[11px] text-fg-subtle lg:px-8">
          {serverConfig.branding.footerUrl ? (
            <a
              href={serverConfig.branding.footerUrl}
              target="_blank"
              rel="noreferrer noopener"
              className="transition-colors hover:text-fg-muted"
            >
              {serverConfig.branding.footerText}
            </a>
          ) : (
            serverConfig.branding.footerText
          )}
        </footer>
      </div>
    </div>
  )
}

/**
 * Instance mark and name, both admin-configurable.
 *
 * The logo is an <img> rather than inlined markup: it is an admin-supplied source
 * (url or data uri), and an image element cannot execute what it loads.
 */
function Brand() {
  const { instanceName, logoUrl } = serverConfig.branding

  return (
    <span className="flex min-w-0 items-center gap-2">
      {logoUrl ? (
        <img src={logoUrl} alt="" className="h-5 w-5 shrink-0 object-contain" />
      ) : (
        <span
          aria-hidden="true"
          className="inline-block h-5 w-5 shrink-0 rounded-full border-2 border-accent border-r-transparent"
        />
      )}
      <span className="truncate text-sm font-semibold tracking-tight text-fg">{instanceName}</span>
    </span>
  )
}
