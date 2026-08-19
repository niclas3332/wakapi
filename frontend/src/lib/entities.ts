/**
 * Heartbeats carry the absolute path of the edited file, so an entity key reads
 * like /Users/me/projects/wakapi/routes/api/ui/summary.go. Printed in full it
 * truncates from the right and hides the only part that identifies the file, which
 * is why every entity label shows the file name and keeps the path as a tooltip.
 */

/** File name without its directories, for both posix and windows separators. */
export function fileName(path: string): string {
  if (!path) return path
  const cut = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  const name = cut === -1 ? path : path.slice(cut + 1)
  return name || path
}

/** The directory a file sits in, for disambiguating two files of the same name. */
export function parentDir(path: string): string {
  if (!path) return ''
  const cut = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  if (cut <= 0) return ''
  const dir = path.slice(0, cut)
  const parentCut = Math.max(dir.lastIndexOf('/'), dir.lastIndexOf('\\'))
  return parentCut === -1 ? dir : dir.slice(parentCut + 1)
}

/** Whether a key looks like a file path rather than a plain name. */
export function isPath(key: string): boolean {
  return key.includes('/') || key.includes('\\')
}
