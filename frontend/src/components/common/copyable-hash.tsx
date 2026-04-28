import { useEffect, useRef, useState } from "react"

import { truncateHash } from "@/lib/format"
import { cn } from "@/lib/utils"

// CopyableHash renders a content-hash chip that copies the full,
// untruncated value to the clipboard when clicked. Use in detail-page
// footers where the hash is informational but the full digest is
// occasionally useful (e.g. comparing against git or another tool).
//
// Feedback is in-place — the chip swaps its text to "Copied" for ~1.4s
// instead of firing a toast, since hash copies are quiet, frequent
// actions where a global notification would be noise.
export function CopyableHash({
  hash,
  className,
}: {
  hash: string
  className?: string
}) {
  const [state, setState] = useState<"idle" | "copied" | "error">("idle")
  const timer = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (timer.current !== null) window.clearTimeout(timer.current)
    }
  }, [])

  async function copy() {
    if (timer.current !== null) window.clearTimeout(timer.current)
    try {
      await navigator.clipboard.writeText(hash)
      setState("copied")
    } catch {
      setState("error")
    }
    timer.current = window.setTimeout(() => setState("idle"), 1400)
  }

  const label =
    state === "copied"
      ? "Copied"
      : state === "error"
        ? "Copy failed"
        : `Hash: ${truncateHash(hash)}`

  return (
    <button
      type="button"
      onClick={copy}
      title={hash}
      aria-label={`Copy full hash ${hash} to clipboard`}
      aria-live="polite"
      className={cn(
        "inline-flex w-fit cursor-pointer items-center text-[0.6875rem] text-muted-foreground hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
        state === "idle" ? "font-mono" : "font-sans",
        state === "copied" && "text-foreground",
        state === "error" && "text-destructive",
        className
      )}
    >
      {label}
    </button>
  )
}
