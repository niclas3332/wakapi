// Config injected by the Go shell handler in place of the `<!--wakapi:head-->`
// marker. Contains nothing user-specific — personal data comes from
// `GET /api/ui/bootstrap` instead.
export type ServerConfig = {
  basePath: string
  version: string
  publicUrl: string
  dbType: string
  supportContact: string
  avatarUrlTemplate: string
  dataRetentionMonths: number
  features: {
    leaderboardEnabled: boolean
    leaderboardRequireAuth: boolean
    invitesEnabled: boolean
    subscriptionsEnabled: boolean
    importEnabled: boolean
    webauthnEnabled: boolean
    allowSignup: boolean
  }
}

declare global {
  interface Window {
    __WAKAPI__?: ServerConfig
  }
}

const DEV_FALLBACK: ServerConfig = {
  basePath: '',
  version: 'dev',
  publicUrl: 'http://localhost:3001',
  dbType: 'unknown',
  supportContact: '',
  avatarUrlTemplate: 'api/avatar/{username_hash}.svg',
  dataRetentionMonths: -1,
  features: {
    leaderboardEnabled: true,
    leaderboardRequireAuth: false,
    invitesEnabled: true,
    subscriptionsEnabled: false,
    importEnabled: true,
    webauthnEnabled: true,
    allowSignup: true,
  },
}

// Read once at module load. The router basename must be known synchronously,
// which is why this is injected into the HTML rather than fetched.
export const serverConfig: ServerConfig = window.__WAKAPI__ ?? DEV_FALLBACK

if (!window.__WAKAPI__ && !import.meta.env.DEV) {
  // eslint-disable-next-line no-console
  console.error('wakapi: server config missing — the shell handler did not inject it')
}

/** Absolute path below the deployment base path, e.g. basePath("/api/ui/summary"). */
export function basePath(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  return `${serverConfig.basePath}${p}`
}
