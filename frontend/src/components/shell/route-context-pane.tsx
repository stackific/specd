import { useEffect, useRef, useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import type { SearchResult, SearchResultKind } from "@/lib/api/search"
import {
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInput,
  useSidebar,
} from "@/components/ui/sidebar"
import { Badge } from "@/components/ui/badge"
import { detailPathFor, kindLabel, searchAll } from "@/lib/api/search"

type FetchState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "ok"; results: Array<SearchResult> }
  | { status: "error"; message: string }

const PAGE_SIZE = 20

export function RouteContextPane() {
  const navigate = useNavigate()
  const [query, setQuery] = useState("")
  const [state, setState] = useState<FetchState>({ status: "idle" })
  const inputRef = useRef<HTMLInputElement | null>(null)
  const { state: sidebarState } = useSidebar()

  // Auto-focus the search input whenever the sidebar opens. The pane stays
  // mounted in both states (CSS toggles visibility via the parent), so a
  // ref-based focus call is enough — no remount needed. A short timeout
  // lets the open animation start before focus, which avoids the focus
  // ring flashing on a still-hidden element.
  useEffect(() => {
    if (sidebarState !== "expanded") return
    const t = window.setTimeout(() => {
      inputRef.current?.focus()
      inputRef.current?.select()
    }, 50)
    return () => window.clearTimeout(t)
  }, [sidebarState])

  useEffect(() => {
    const trimmed = query.trim()
    if (!trimmed) {
      setState({ status: "idle" })
      return
    }
    const ctrl = new AbortController()
    const timer = setTimeout(() => {
      setState({ status: "loading" })
      searchAll({
        q: trimmed,
        kind: "all",
        page: 1,
        pageSize: PAGE_SIZE,
        signal: ctrl.signal,
      })
        .then((res) => {
          if (ctrl.signal.aborted) return
          setState({ status: "ok", results: res.items })
        })
        .catch((err: unknown) => {
          if (ctrl.signal.aborted) return
          const message =
            err instanceof Error ? err.message : "Search unavailable"
          setState({ status: "error", message })
        })
    }, 200)
    return () => {
      clearTimeout(timer)
      ctrl.abort()
    }
  }, [query])

  function open(kind: SearchResultKind, id: string) {
    navigate({ to: detailPathFor(kind, id) })
  }

  return (
    <>
      <SidebarHeader className="gap-3.5 border-b p-4">
        <div className="text-base font-medium text-foreground">
          Quick Search
        </div>
        <SidebarInput
          ref={inputRef}
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search specs, tasks, KB…"
          aria-label="Search specs, tasks, and knowledge base"
          className="pr-7 [&::-webkit-search-cancel-button]:mr-1"
        />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup className="px-0">
          <SidebarGroupContent aria-live="polite">
            <ResultsBody state={state} query={query} onOpen={open} />
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </>
  )
}

function ResultsBody({
  state,
  query,
  onOpen,
}: {
  state: FetchState
  query: string
  onOpen: (kind: SearchResultKind, id: string) => void
}) {
  if (state.status === "idle") {
    return (
      <p className="px-4 py-6 text-sm text-muted-foreground">
        Type to search across specs, tasks, and KB.
      </p>
    )
  }

  if (state.status === "loading") {
    return <p className="px-4 py-6 text-sm text-muted-foreground">Searching…</p>
  }

  if (state.status === "error") {
    return <p className="px-4 py-6 text-sm text-destructive">{state.message}</p>
  }

  if (state.results.length === 0) {
    return (
      <p className="px-4 py-6 text-sm text-muted-foreground">
        No results for &ldquo;{query.trim()}&rdquo;.
      </p>
    )
  }

  return (
    <ul role="list" className="flex flex-col">
      {state.results.map((r) => (
        <li role="listitem" key={`${r.kind}-${r.id}`}>
          <button
            type="button"
            onClick={() => onOpen(r.kind, r.id)}
            className="flex w-full flex-col items-start gap-1 border-b p-4 text-left text-sm leading-tight whitespace-nowrap last:border-b-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <div className="flex w-full items-center gap-2">
              <Badge variant="secondary" className="text-xs">
                {kindLabel(r.kind)}
              </Badge>
              <span className="ml-auto font-mono text-xs text-muted-foreground">
                {r.id}
              </span>
            </div>
            <span className="font-medium">{r.title}</span>
            {r.summary ? (
              <span className="line-clamp-2 w-[260px] text-xs whitespace-break-spaces text-muted-foreground">
                {r.summary}
              </span>
            ) : null}
          </button>
        </li>
      ))}
    </ul>
  )
}
