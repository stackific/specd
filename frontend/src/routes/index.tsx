import { createFileRoute, redirect } from "@tanstack/react-router"
import { getMeta } from "@/lib/api/meta"

const FALLBACK = "/welcome"

export const Route = createFileRoute("/")({
  beforeLoad: async () => {
    let target: string = FALLBACK
    try {
      const meta = await getMeta()
      if (meta.default_route) target = meta.default_route
    } catch {
      // Meta fetch failed — fall back to /welcome.
    }
    throw redirect({ to: target as never })
  },
})
