import { serverConfig } from '@/lib/bootstrap'

/**
 * Header the server requires on every mutation. A cross-origin form cannot set
 * headers and a cross-origin fetch that tries triggers a preflight the server
 * never answers, which is what makes this a csrf defence.
 */
const CSRF_HEADER = 'X-Wakapi-Request'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/**
 * Build an absolute url below the deployment base path.
 *
 * Never hand-write `fetch('/api/...')` or `fetch('api/...')`: the document carries
 * `<base href="{basePath}/assets/app/">`, so a relative url would resolve into the
 * asset directory, and an absolute one ignores base_path deployments.
 */
export function apiUrl(path: string, params?: URLSearchParams | Record<string, string>): string {
  const clean = path.startsWith('/') ? path : `/${path}`
  const search = params instanceof URLSearchParams ? params : new URLSearchParams(params ?? {})
  const query = search.toString()
  return `${serverConfig.basePath}/api/ui${clean}${query ? `?${query}` : ''}`
}

/** Path below the deployment base path, for links and non-api resources. */
export function appUrl(path: string): string {
  return `${serverConfig.basePath}${path.startsWith('/') ? path : `/${path}`}`
}

type RequestOptions = {
  method?: string
  body?: unknown
  params?: URLSearchParams | Record<string, string>
  signal?: AbortSignal
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, params, signal } = options

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (method !== 'GET' && method !== 'HEAD') {
    headers[CSRF_HEADER] = '1'
    if (body !== undefined) headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(apiUrl(path, params), {
    method,
    headers,
    credentials: 'same-origin',
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  })

  if (response.status === 401) {
    // The session expired mid-visit. The login page is server-rendered, so this is a
    // full navigation rather than a client-side route change.
    window.location.assign(appUrl('/login'))
    throw new ApiError(401, 'unauthorized')
  }

  if (!response.ok) {
    throw new ApiError(response.status, await readError(response))
  }

  if (response.status === 204) return undefined as T
  return (await response.json()) as T
}

async function readError(response: Response): Promise<string> {
  try {
    const data = await response.json()
    if (data && typeof data.error === 'string') return data.error
  } catch {
    // fall through to the status text
  }
  return response.statusText || `request failed with status ${response.status}`
}
