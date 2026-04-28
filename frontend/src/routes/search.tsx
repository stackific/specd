import { useEffect, useMemo, useRef, useState } from "react"
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router"
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react"
import type { SearchKind, SearchResponse, SearchResult } from "@/lib/api/search"
import { detailPathFor, kindLabel, searchAll } from "@/lib/api/search"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Pagination,
  PaginationContent,
  PaginationItem,
} from "@/components/ui/pagination"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

type SearchParams = {
  q?: string
  kind?: SearchKind
  page?: number
  page_size?: number
}

const DEFAULT_PAGE_SIZE = 10

export const Route = createFileRoute("/search")({
  validateSearch: (search: Record<string, unknown>): SearchParams => ({
    q: typeof search.q === "string" ? search.q : undefined,
    kind:
      search.kind === "spec" ||
      search.kind === "task" ||
      search.kind === "kb" ||
      search.kind === "all"
        ? search.kind
        : undefined,
    page: typeof search.page === "number" ? search.page : undefined,
    page_size:
      typeof search.page_size === "number" ? search.page_size : undefined,
  }),
  component: SearchPage,
})

function SearchPage() {
  const params = Route.useSearch()
  const navigate = useNavigate()

  const q = params.q ?? ""
  const kind: SearchKind = params.kind ?? "all"
  const page = params.page ?? 1
  const pageSize = params.page_size ?? DEFAULT_PAGE_SIZE

  // Local input state, debounced into URL.
  const [input, setInput] = useState(q)
  const inputRef = useRef<HTMLInputElement | null>(null)

  // Keep input in sync if URL changes externally (e.g., palette navigation).
  useEffect(() => {
    setInput(q)
  }, [q])

  // Debounced URL update.
  useEffect(() => {
    if (input === q) return
    const t = setTimeout(() => {
      navigate({
        to: "/search",
        search: (prev) => ({
          ...prev,
          q: input || undefined,
          page: 1,
        }),
      })
    }, 300)
    return () => clearTimeout(t)
  }, [input, q, navigate])

  // Fetch state.
  const [data, setData] = useState<SearchResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!q.trim()) {
      setData(null)
      setLoading(false)
      setError(null)
      return
    }
    const ctrl = new AbortController()
    setLoading(true)
    setError(null)
    searchAll({ q, kind, page, pageSize, signal: ctrl.signal })
      .then((res) => {
        setData(res)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (ctrl.signal.aborted) return
        setLoading(false)
        setError(err instanceof Error ? err.message : "Search failed")
      })
    return () => ctrl.abort()
  }, [q, kind, page, pageSize])

  function onTabChange(next: string) {
    navigate({
      to: "/search",
      search: (prev) => ({
        ...prev,
        kind: next as SearchKind,
        page: 1,
      }),
    })
  }

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    navigate({
      to: "/search",
      search: (prev) => ({
        ...prev,
        q: input || undefined,
        page: 1,
      }),
    })
  }

  return (
    <div className="container mx-auto max-w-4xl space-y-6 p-6">
      <header className="space-y-4">
        <h1 className="text-3xl font-semibold tracking-tight">Search</h1>
        <form role="search" onSubmit={onSubmit}>
          <Input
            ref={inputRef}
            type="search"
            autoFocus
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Search specs, tasks, and KB..."
            aria-label="Search specs, tasks, and KB"
            className="h-11 w-full text-base"
          />
        </form>
        <Tabs value={kind} onValueChange={onTabChange}>
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="spec">Specs</TabsTrigger>
            <TabsTrigger value="task">Tasks</TabsTrigger>
            <TabsTrigger value="kb">KB</TabsTrigger>
          </TabsList>
        </Tabs>
      </header>

      <section aria-live="polite" className="space-y-4">
        <ResultsArea q={q} loading={loading} error={error} data={data} />
      </section>

      {data && data.total_pages > 1 ? (
        <ResultsPagination
          page={data.page}
          totalPages={data.total_pages}
          q={q}
          kind={kind}
          pageSize={pageSize}
        />
      ) : null}
    </div>
  )
}

function ResultsArea({
  q,
  loading,
  error,
  data,
}: {
  q: string
  loading: boolean
  error: string | null
  data: SearchResponse | null
}) {
  if (!q.trim()) {
    return (
      <p className="text-sm text-muted-foreground">
        Type to search across specs, tasks, and KB.
      </p>
    )
  }
  if (loading && !data) {
    return <ResultsSkeleton />
  }
  if (error) {
    return (
      <p className="text-sm text-destructive" role="alert">
        {error}
      </p>
    )
  }
  if (!data || data.total_count === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No results for &ldquo;{q}&rdquo;.
      </p>
    )
  }

  return (
    <>
      <p className="text-sm text-muted-foreground">
        {data.total_count} result{data.total_count === 1 ? "" : "s"} for &ldquo;
        {q}&rdquo;.
      </p>
      <ResultGroups data={data} />
    </>
  )
}

function ResultGroups({ data }: { data: SearchResponse }) {
  // For "all" kind, render grouped sections; otherwise the flat items array.
  const grouped = useMemo(
    () => ({
      specs: data.page_specs,
      tasks: data.page_tasks,
      kb: data.page_kb,
    }),
    [data]
  )
  const isAll = data.kind === "all"

  if (!isAll) {
    return (
      <section aria-label="Results" className="space-y-3">
        {data.items.map((r) => (
          <ResultCard key={`${r.kind}-${r.id}`} result={r} />
        ))}
      </section>
    )
  }

  return (
    <>
      {grouped.specs.length > 0 ? (
        <section aria-label="Spec results" className="space-y-3">
          <h2 className="text-sm font-medium text-muted-foreground">Specs</h2>
          {grouped.specs.map((r) => (
            <ResultCard key={`spec-${r.id}`} result={r} />
          ))}
        </section>
      ) : null}
      {grouped.tasks.length > 0 ? (
        <section aria-label="Task results" className="space-y-3">
          <h2 className="text-sm font-medium text-muted-foreground">Tasks</h2>
          {grouped.tasks.map((r) => (
            <ResultCard key={`task-${r.id}`} result={r} />
          ))}
        </section>
      ) : null}
      {grouped.kb.length > 0 ? (
        <section aria-label="KB results" className="space-y-3">
          <h2 className="text-sm font-medium text-muted-foreground">KB</h2>
          {grouped.kb.map((r) => (
            <ResultCard key={`kb-${r.id}`} result={r} />
          ))}
        </section>
      ) : null}
    </>
  )
}

function ResultCard({ result }: { result: SearchResult }) {
  const kind = result.kind
  const path = detailPathFor(kind, result.id)
  return (
    <Card>
      <CardHeader className="space-y-2">
        <div className="flex items-center gap-2">
          <Badge variant="secondary">{kindLabel(kind)}</Badge>
          <span className="text-xs text-muted-foreground">{result.id}</span>
          <span className="ml-auto font-mono text-xs text-muted-foreground">
            {result.score.toFixed(2)}
          </span>
        </div>
        <CardTitle className="text-base leading-snug break-words">
          <Link to={path} className="hover:underline">
            {result.title}
          </Link>
        </CardTitle>
      </CardHeader>
      {result.summary ? (
        <CardContent className="text-sm break-words text-muted-foreground">
          {result.summary}
        </CardContent>
      ) : null}
    </Card>
  )
}

function ResultsSkeleton() {
  return (
    <div className="space-y-3" aria-label="Loading results">
      {[0, 1, 2].map((i) => (
        <div key={i} className="space-y-2 rounded-xl border p-4">
          <div className="flex items-center gap-2">
            <Skeleton className="h-5 w-12" />
            <Skeleton className="h-4 w-16" />
          </div>
          <Skeleton className="h-5 w-3/4" />
          <Skeleton className="h-4 w-full" />
        </div>
      ))}
    </div>
  )
}

function ResultsPagination({
  page,
  totalPages,
  q,
  kind,
  pageSize,
}: {
  page: number
  totalPages: number
  q: string
  kind: SearchKind
  pageSize: number
}) {
  const pages = pageWindow(page, totalPages)
  const baseSearch = (target: number): SearchParams => ({
    q,
    kind,
    page: target,
    page_size: pageSize === DEFAULT_PAGE_SIZE ? undefined : pageSize,
  })

  const prevDisabled = page <= 1
  const nextDisabled = page >= totalPages

  return (
    <Pagination>
      <PaginationContent>
        <PaginationItem>
          {prevDisabled ? (
            <span
              aria-disabled
              className={cn(
                buttonVariants({ variant: "ghost", size: "default" }),
                "pointer-events-none pl-1.5 opacity-50"
              )}
            >
              <ChevronLeftIcon data-icon="inline-start" />
              <span className="hidden sm:block">Previous</span>
            </span>
          ) : (
            <Link
              to="/search"
              search={baseSearch(page - 1)}
              aria-label="Go to previous page"
              className={cn(
                buttonVariants({ variant: "ghost", size: "default" }),
                "pl-1.5"
              )}
            >
              <ChevronLeftIcon data-icon="inline-start" />
              <span className="hidden sm:block">Previous</span>
            </Link>
          )}
        </PaginationItem>
        {pages.map((p) => (
          <PaginationItem key={p} className="hidden sm:block">
            <Link
              to="/search"
              search={baseSearch(p)}
              aria-current={p === page ? "page" : undefined}
              className={cn(
                buttonVariants({
                  variant: p === page ? "outline" : "ghost",
                  size: "icon",
                })
              )}
            >
              {p}
            </Link>
          </PaginationItem>
        ))}
        <PaginationItem>
          {nextDisabled ? (
            <span
              aria-disabled
              className={cn(
                buttonVariants({ variant: "ghost", size: "default" }),
                "pointer-events-none pr-1.5 opacity-50"
              )}
            >
              <span className="hidden sm:block">Next</span>
              <ChevronRightIcon data-icon="inline-end" />
            </span>
          ) : (
            <Link
              to="/search"
              search={baseSearch(page + 1)}
              aria-label="Go to next page"
              className={cn(
                buttonVariants({ variant: "ghost", size: "default" }),
                "pr-1.5"
              )}
            >
              <span className="hidden sm:block">Next</span>
              <ChevronRightIcon data-icon="inline-end" />
            </Link>
          )}
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}

function pageWindow(current: number, total: number): Array<number> {
  const span = 5
  let start = Math.max(1, current - 2)
  const end = Math.min(total, start + span - 1)
  if (end - start + 1 < span) {
    start = Math.max(1, end - span + 1)
  }
  const out: Array<number> = []
  for (let i = start; i <= end; i++) out.push(i)
  return out
}
