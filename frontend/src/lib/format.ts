// Shared formatting helpers used across detail pages and lists.

// formatDateTime renders an ISO timestamp with date plus hour/minute
// (e.g. "Apr 28, 2026, 10:23 PM"). Use this in detail-page footers where
// the precise modification time matters.
export function formatDateTime(iso: string): string {
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

// Thresholds, in seconds, that gate the unit picker in formatRelativeTime.
// Each threshold holds [upper bound exclusive, divisor, Intl unit]. Walk the
// list top-to-bottom and use the first row whose bound the elapsed seconds
// fall under. Past those, we fall through to "years".
const RELATIVE_THRESHOLDS: ReadonlyArray<
  [number, number, Intl.RelativeTimeFormatUnit]
> = [
  [60, 1, "second"],
  [3600, 60, "minute"],
  [86_400, 3600, "hour"],
  [604_800, 86_400, "day"],
  [2_592_000, 604_800, "week"], // ~30 days
  [31_536_000, 2_592_000, "month"], // ~365 days
]

// formatRelativeTime renders an ISO timestamp as a human-friendly relative
// string, e.g. "3 minutes ago", "2 days ago", "in 4 hours". Pair it with
// the absolute timestamp in `title` for the hover tooltip — the relative
// form is for skim-reading, not for picking a time out of a list.
//
// Uses `Intl.RelativeTimeFormat` for plural correctness across locales.
// `nowSec` is injectable so tests can pin "now" without faking timers.
export function formatRelativeTime(
  iso: string,
  nowMs: number = Date.now()
): string {
  if (!iso) return ""
  const d = new Date(iso)
  const t = d.getTime()
  if (Number.isNaN(t)) return iso

  const deltaSec = (t - nowMs) / 1000
  const absSec = Math.abs(deltaSec)
  if (absSec < 30) return "just now"

  const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" })
  for (const [bound, divisor, unit] of RELATIVE_THRESHOLDS) {
    if (absSec < bound) {
      return fmt.format(Math.round(deltaSec / divisor), unit)
    }
  }
  // > 1 year: render in years.
  return fmt.format(Math.round(deltaSec / 31_536_000), "year")
}

// Truncates a long content hash to first-8 + … + last-8 characters so it
// stays readable in tight footer rows. Use the full hash in `title` for the
// hover tooltip.
export function truncateHash(hash: string): string {
  if (hash.length <= 16) return hash
  return `${hash.slice(0, 8)}…${hash.slice(-8)}`
}

// Converts a status / type slug ("in_progress", "wont_fix", "nonfunctional")
// into a human-readable label ("In progress", "Wont fix", "Nonfunctional"):
// lowercase underscored and dashed slugs become space-separated, with the
// first letter uppercased. Mirrors the Go `FromSlug` helper.
export function humanizeSlug(slug: string): string {
  if (!slug) return ""
  const spaced = slug.replace(/[_-]+/g, " ").trim()
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}
