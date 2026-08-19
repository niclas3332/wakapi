import { createBrowserRouter, Navigate } from 'react-router'

import { serverConfig } from '@/lib/bootstrap'
import { Layout } from './Layout'
import { SummaryPage } from '@/pages/Summary'
import { ProjectsPage } from '@/pages/Projects'
import { LeaderboardPage } from '@/pages/Leaderboard'
import { SettingsPage } from '@/pages/Settings'
import { InsightsPage } from '@/pages/Insights'
import { WrappedPage } from '@/pages/Wrapped'
import { AiOverviewPage } from '@/pages/AiOverview'

export const router = createBrowserRouter(
  [
    {
      element: <Layout />,
      children: [
        { index: true, element: <Navigate to="/summary" replace /> },
        { path: 'summary', element: <SummaryPage /> },
        { path: 'projects', element: <ProjectsPage /> },
        { path: 'leaderboard', element: <LeaderboardPage /> },
        { path: 'settings', element: <SettingsPage /> },
        { path: 'insights', element: <InsightsPage /> },
        { path: 'wrapped', element: <WrappedPage /> },
        { path: 'ai', element: <AiOverviewPage /> },
      ],
    },
  ],
  {
    // The server owns the mount point. It differs from basePath while the app is
    // staged alongside the legacy pages, and becomes equal to it at cutover.
    basename: serverConfig.appBase || '/',
  },
)
