import { describe, expect, it } from "vitest"

import {
  formatDateTime,
  formatRelativeTime,
  humanizeSlug,
  truncateHash,
} from "./format"

// Pinned "now" so the tests are deterministic regardless of when they run.
// 2026-04-28T22:00:00Z — matches the seed-data ballpark in tmp/qa/.
const NOW_MS = Date.parse("2026-04-28T22:00:00Z")

// secondsAgo returns an ISO timestamp `n` seconds before NOW_MS, used by the
// table-driven tests below to keep the inputs readable.
function secondsAgo(n: number): string {
  return new Date(NOW_MS - n * 1000).toISOString()
}

// secondsFromNow is the future variant — tests that "in X minutes" and the
// like render correctly when the timestamp is ahead of `now`.
function secondsFromNow(n: number): string {
  return new Date(NOW_MS + n * 1000).toISOString()
}

describe("formatRelativeTime", () => {
  // "just now" boundary: < 30 seconds in either direction.
  it("renders sub-30-second deltas as 'just now'", () => {
    expect(formatRelativeTime(secondsAgo(0), NOW_MS)).toBe("just now")
    expect(formatRelativeTime(secondsAgo(15), NOW_MS)).toBe("just now")
    expect(formatRelativeTime(secondsFromNow(20), NOW_MS)).toBe("just now")
  })

  // Past-tense bucket walk. We use Intl.RelativeTimeFormat, so the exact
  // wording can vary by locale; assert the unit is right and the magnitude
  // is sensible rather than locking to a single English string.
  it("picks the right unit for past timestamps", () => {
    // Just past the "just now" threshold but still under a minute.
    expect(formatRelativeTime(secondsAgo(45), NOW_MS)).toMatch(/second/)
    // Multiple minutes.
    expect(formatRelativeTime(secondsAgo(5 * 60), NOW_MS)).toMatch(
      /5\s*minutes/
    )
    // Hours.
    expect(formatRelativeTime(secondsAgo(3 * 3600), NOW_MS)).toMatch(
      /3\s*hours/
    )
    // Days.
    expect(formatRelativeTime(secondsAgo(2 * 86_400), NOW_MS)).toMatch(
      /2\s*days/
    )
    // Weeks (>= 7 days, < 30 days).
    expect(formatRelativeTime(secondsAgo(10 * 86_400), NOW_MS)).toMatch(/week/)
    // Months (>= 30 days, < 365 days).
    expect(formatRelativeTime(secondsAgo(60 * 86_400), NOW_MS)).toMatch(/month/)
    // Years.
    expect(formatRelativeTime(secondsAgo(2 * 365 * 86_400), NOW_MS)).toMatch(
      /year/
    )
  })

  // Future-tense bucket. Same unit detection, but "in" prefix.
  it("renders future timestamps with 'in' prefix", () => {
    const out = formatRelativeTime(secondsFromNow(5 * 60), NOW_MS)
    expect(out.toLowerCase()).toContain("in")
    expect(out).toMatch(/minute/)
  })

  // Defensive cases.
  it("returns empty string for empty input", () => {
    expect(formatRelativeTime("", NOW_MS)).toBe("")
  })

  it("returns the raw input for un-parseable timestamps", () => {
    expect(formatRelativeTime("not-a-date", NOW_MS)).toBe("not-a-date")
  })
})

describe("formatDateTime", () => {
  it("renders an ISO timestamp with date and time", () => {
    const out = formatDateTime("2026-04-28T22:00:00Z")
    // Year + a recognisable hour — exact format is locale-dependent so we
    // assert structure instead of locking the entire string.
    expect(out).toMatch(/2026/)
    expect(out).toMatch(/[0-9]:[0-9]{2}/)
  })

  it("returns empty string for empty input", () => {
    expect(formatDateTime("")).toBe("")
  })
})

describe("truncateHash", () => {
  it("truncates a long sha256 with an ellipsis", () => {
    const hash = "a".repeat(8) + "b".repeat(48) + "c".repeat(8)
    const out = truncateHash(hash)
    expect(out).toBe("aaaaaaaa…cccccccc")
  })

  it("returns the input unchanged when it is short", () => {
    expect(truncateHash("abc")).toBe("abc")
  })
})

describe("humanizeSlug", () => {
  it("space-separates and capitalises the first letter", () => {
    expect(humanizeSlug("in_progress")).toBe("In progress")
    expect(humanizeSlug("wont_fix")).toBe("Wont fix")
    expect(humanizeSlug("nonfunctional")).toBe("Nonfunctional")
    expect(humanizeSlug("")).toBe("")
  })
})
