import type { AvailableFilters, Summary, SummaryResponse } from './types'

/**
 * Go marshals a nil slice as `null`, not `[]`.
 *
 * The summary endpoint does this for `branches` and `entities`, which are only
 * computed when a project filter is active, and can do it for any other collection
 * that happens to be empty. Normalising once at the boundary keeps every component
 * free of null checks - and keeps the public /api/summary contract untouched, since
 * this is a client-side concern.
 */
function list<T>(value: T[] | null | undefined): T[] {
  return value ?? []
}

export function normalizeSummary(summary: Summary): Summary {
  return {
    ...summary,
    projects: list(summary.projects),
    languages: list(summary.languages),
    editors: list(summary.editors),
    operating_systems: list(summary.operating_systems),
    machines: list(summary.machines),
    labels: list(summary.labels),
    branches: list(summary.branches),
    entities: list(summary.entities),
    categories: list(summary.categories),
  }
}

function normalizeFilters(filters: AvailableFilters): AvailableFilters {
  return {
    projects: list(filters?.projects),
    languages: list(filters?.languages),
    machines: list(filters?.machines),
    labels: list(filters?.labels),
    categories: list(filters?.categories),
  }
}

export function normalizeSummaryResponse(response: SummaryResponse): SummaryResponse {
  return {
    ...response,
    summary: normalizeSummary(response.summary),
    available_filters: normalizeFilters(response.available_filters),
    editor_colors: response.editor_colors ?? {},
    language_colors: response.language_colors ?? {},
    os_colors: response.os_colors ?? {},
  }
}
