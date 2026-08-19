/**
 * Categorical series colours, in fixed order.
 *
 * Validated as a set against the card surface (#111823) with the data-viz palette
 * checker: lightness band, chroma floor, adjacent-pair CVD separation, normal-vision
 * floor and 3:1 contrast all pass. Do not reorder and do not append a ninth hue -
 * anything past the eighth series folds into "Other".
 */
export const SERIES_COLORS = [
  '#3987e5', // blue
  '#d95926', // orange
  '#199e70', // aqua
  '#c98500', // yellow
  '#d55181', // magenta
  '#008300', // green
  '#9085e9', // violet
  '#e66767', // red
] as const

export const MAX_SERIES = SERIES_COLORS.length

/** Blue is the machine, green is the person - one meaning app-wide. */
export const AI_COLOR = '#3987e5'
export const HUMAN_COLOR = '#199e70'

/** Neutral for the folded "Other" bucket and for unknown keys. */
export const OTHER_COLOR = '#4a5a6e'

export const UNKNOWN_KEY = 'unknown'
export const OTHER_KEY = 'Other'

/**
 * Resolve the colour for one entity.
 *
 * Wakapi ships an identity colour per language, editor and operating system
 * (data/colors.json, surfaced by the api). Those win, because users read
 * "JavaScript is yellow" as meaning. Everything else - projects, machines, labels,
 * categories, branches - gets a series slot by position, so a colour follows the
 * entity rather than its rank.
 */
export function colorFor(key: string, index: number, identityColors?: Record<string, string>): string {
  if (key === OTHER_KEY) return OTHER_COLOR
  const identity = identityColors?.[key.toLowerCase()]
  if (identity) return identity
  if (index >= MAX_SERIES) return OTHER_COLOR
  return SERIES_COLORS[index]
}

/**
 * Pick a series slot from the name itself, so an entity keeps its colour no matter
 * where it lands - across pages, filters and reloads.
 *
 * Ranked lists deliberately do not use this: there, position carries meaning and
 * every row is labelled, so colour is a magnitude cue rather than an identity one.
 * Use this wherever the same entity can appear in different places.
 */
export function stableColorFor(key: string): string {
  let hash = 0
  for (let i = 0; i < key.length; i++) {
    hash = (hash * 31 + key.charCodeAt(i)) | 0
  }
  return SERIES_COLORS[Math.abs(hash) % MAX_SERIES]
}

export type Sliced<T> = {
  shown: T[]
  otherSeconds: number
  otherCount: number
}

/**
 * Take the top n entries and fold the remainder into a single "Other" bucket, so a
 * long tail never forces invented hues onto the chart.
 */
export function sliceTop<T extends { key: string; seconds: number }>(
  items: T[],
  limit: number,
): Sliced<T> {
  if (items.length <= limit) return { shown: items, otherSeconds: 0, otherCount: 0 }
  const shown = items.slice(0, limit)
  const rest = items.slice(limit)
  return {
    shown,
    otherSeconds: rest.reduce((sum, item) => sum + item.seconds, 0),
    otherCount: rest.length,
  }
}
