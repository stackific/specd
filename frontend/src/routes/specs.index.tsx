import { useEffect, useState } from "react"
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router"
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react"
import type { ListSpecsResponse, SpecsView } from "@/lib/api/specs"
import { SPECS_TYPE_ALL, listSpecs } from "@/lib/api/specs"
import { ApiError } from "@/lib/api"
import { formatDateTime, formatRelativeTime } from "@/lib/format"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
} from "@/components/ui/pagination"

type SpecsSearch = {
  view?: SpecsView
  type?: string
  page?: number
  page_size?: number
}

type ResolvedSpecsSearch = {
  view: SpecsView
  type: string
  page: number
  page_size: number
}

const VALID_VIEWS: ReadonlyArray<SpecsView> = ["grouped", "cards", "flat"]

function isSpecsView(v: unknown): v is SpecsView {
  return (
    typeof v === "string" && (VALID_VIEWS as ReadonlyArray<string>).includes(v)
  )
}

export const Route = createFileRoute("/specs/")({
  validateSearch: (search: Record<string, unknown>): SpecsSearch => {
    const out: SpecsSearch = {}
    if (isSpecsView(search.view)) out.view = search.view
    if (typeof search.type === "string" && search.type.length > 0)
      out.type = search.type
    if (typeof search.page === "number" && search.page > 0)
      out.page = search.page
    if (typeof search.page_size === "number" && search.page_size > 0) {
      out.page_size = search.page_size
    }
    return out
  },
  component: SpecsListPage,
})

function resolveSearch(s: SpecsSearch): ResolvedSpecsSearch {
  return {
    view: s.view ?? "grouped",
    type: s.type ?? SPECS_TYPE_ALL,
    page: s.page ?? 1,
    page_size: s.page_size ?? 20,
  }
}

function SpecsListPage() {
  const rawSearch = Route.useSearch()
  const search = resolveSearch(rawSearch)
  const navigate = useNavigate({ from: Route.fullPath })
  const [data, setData] = useState<ListSpecsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError(null)
    listSpecs({
      view: search.view,
      type: search.type,
      page: search.page,
      pageSize: search.page_size,
      signal: controller.signal,
    })
      .then((res) => {
        setData(res)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        if (err instanceof ApiError) {
          setError(err.message || "Failed to load specs.")
        } else {
          setError("Failed to load specs.")
        }
        setLoading(false)
      })
    return () => controller.abort()
  }, [search.view, search.type, search.page, search.page_size, reloadKey])

  function setView(view: SpecsView) {
    navigate({ search: { ...search, view, page: 1 } })
  }

  function setTypeFilter(type: string) {
    navigate({ search: { ...search, type, page: 1 } })
  }

  return (
    <div className="container mx-auto max-w-6xl space-y-6 p-6">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-semibold tracking-tight">Specs</h1>
          <p className="text-sm text-muted-foreground">
            Browse business, functional, and non-functional specs.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Tabs
            value={search.view}
            onValueChange={(v) => setView(v as SpecsView)}
          >
            <TabsList>
              <TabsTrigger value="grouped">Grouped</TabsTrigger>
              <TabsTrigger value="cards">Cards</TabsTrigger>
              <TabsTrigger value="flat">Flat</TabsTrigger>
            </TabsList>
          </Tabs>
          <TypeFilter
            value={search.type}
            options={data?.types ?? []}
            onChange={setTypeFilter}
          />
        </div>
      </header>

      <section aria-label="Specs list" className="space-y-4">
        {loading ? (
          <SpecsSkeleton view={search.view} />
        ) : error ? (
          <div className="flex items-center gap-3 rounded-md border border-destructive/40 bg-destructive/5 p-4 text-sm">
            <span className="text-destructive">{error}</span>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setReloadKey((k) => k + 1)}
            >
              Retry
            </Button>
          </div>
        ) : !data || data.total_count === 0 ? (
          <Card>
            <CardContent className="p-6 text-sm text-muted-foreground">
              No specs match this filter.
            </CardContent>
          </Card>
        ) : (
          <SpecsView data={data} view={search.view} />
        )}

        {data && data.total_pages > 1 && !loading && !error ? (
          <SpecsPagination search={search} totalPages={data.total_pages} />
        ) : null}
      </section>
    </div>
  )
}

function TypeFilter({
  value,
  options,
  onChange,
}: {
  value: string
  options: Array<string>
  onChange: (v: string) => void
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger aria-label="Filter by type" className="min-w-40">
        <SelectValue placeholder="All types" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={SPECS_TYPE_ALL}>All types</SelectItem>
        {options.map((opt) => (
          <SelectItem key={opt} value={opt} className="capitalize">
            {opt}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function SpecsView({
  data,
  view,
}: {
  data: ListSpecsResponse
  view: SpecsView
}) {
  if (view === "flat") return <FlatView items={data.items} />
  if (view === "cards") return <CardsView items={data.items} />
  return <GroupedView groups={data.groups} />
}

function GroupedView({ groups }: { groups: ListSpecsResponse["groups"] }) {
  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <Card key={group.Type}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-lg capitalize">{group.Type}</CardTitle>
            <Badge variant="secondary" aria-label={`${group.Total} specs`}>
              {group.Total}
            </Badge>
          </CardHeader>
          <CardContent className="p-0">
            <ul className="divide-y">
              {group.Items.map((item) => (
                <li key={item.id}>
                  <Link
                    to="/specs/$id"
                    params={{ id: item.id }}
                    search={{}}
                    className="flex flex-col gap-1 px-6 py-3 hover:bg-accent/50 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="font-mono text-xs text-muted-foreground">
                        {item.id}
                      </span>
                      <span className="truncate font-medium">{item.title}</span>
                    </div>
                    <span className="truncate text-sm text-muted-foreground sm:max-w-md">
                      {item.summary}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function CardsView({ items }: { items: ListSpecsResponse["items"] }) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {items.map((item) => (
        <Link
          key={item.id}
          to="/specs/$id"
          params={{ id: item.id }}
          search={{}}
          className="block focus:outline-none"
        >
          <Card className="h-full transition-colors hover:bg-accent/40">
            <CardHeader className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <span className="font-mono text-xs text-muted-foreground">
                  {item.id}
                </span>
                <Badge variant="outline" className="capitalize">
                  {item.type}
                </Badge>
              </div>
              <CardTitle className="text-base">{item.title}</CardTitle>
              {item.summary ? (
                <CardDescription className="line-clamp-3">
                  {item.summary}
                </CardDescription>
              ) : null}
            </CardHeader>
          </Card>
        </Link>
      ))}
    </div>
  )
}

function FlatView({ items }: { items: ListSpecsResponse["items"] }) {
  return (
    <div className="overflow-x-auto rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-32">ID</TableHead>
            <TableHead>Title</TableHead>
            <TableHead className="w-32">Type</TableHead>
            <TableHead>Summary</TableHead>
            <TableHead className="w-40">Updated</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={item.id} className="cursor-pointer">
              <TableCell className="font-mono text-xs">
                <Link
                  to="/specs/$id"
                  params={{ id: item.id }}
                  search={{}}
                  className="hover:underline"
                >
                  {item.id}
                </Link>
              </TableCell>
              <TableCell className="font-medium">
                <Link
                  to="/specs/$id"
                  params={{ id: item.id }}
                  search={{}}
                  className="hover:underline"
                >
                  {item.title}
                </Link>
              </TableCell>
              <TableCell>
                <Badge variant="outline" className="capitalize">
                  {item.type}
                </Badge>
              </TableCell>
              <TableCell className="max-w-md truncate text-muted-foreground">
                {item.summary}
              </TableCell>
              <TableCell className="text-xs text-muted-foreground">
                <time
                  dateTime={item.updated_at}
                  title={formatDateTime(item.updated_at)}
                >
                  {formatRelativeTime(item.updated_at)}
                </time>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function SpecsSkeleton({ view }: { view: SpecsView }) {
  if (view === "flat") {
    return (
      <div className="space-y-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }
  if (view === "cards") {
    return (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full" />
        ))}
      </div>
    )
  }
  return (
    <div className="space-y-4">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className="h-40 w-full" />
      ))}
    </div>
  )
}

function SpecsPagination({
  search,
  totalPages,
}: {
  search: ResolvedSpecsSearch
  totalPages: number
}) {
  const current = search.page
  const pages = pageWindow(current, totalPages, 5)
  const linkSearch = (page: number): ResolvedSpecsSearch => ({
    ...search,
    page,
  })

  const prevDisabled = current <= 1
  const nextDisabled = current >= totalPages

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
              to="/specs"
              search={linkSearch(current - 1)}
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

        {pages[0] > 1 ? (
          <>
            <PaginationItem>
              <Link
                to="/specs"
                search={linkSearch(1)}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "icon" })
                )}
              >
                1
              </Link>
            </PaginationItem>
            {pages[0] > 2 ? (
              <PaginationItem>
                <PaginationEllipsis />
              </PaginationItem>
            ) : null}
          </>
        ) : null}

        {pages.map((p) => (
          <PaginationItem key={p}>
            <Link
              to="/specs"
              search={linkSearch(p)}
              aria-label={`Page ${p}`}
              aria-current={p === current ? "page" : undefined}
              className={cn(
                buttonVariants({
                  variant: p === current ? "outline" : "ghost",
                  size: "icon",
                })
              )}
            >
              {p}
            </Link>
          </PaginationItem>
        ))}

        {pages[pages.length - 1] < totalPages ? (
          <>
            {pages[pages.length - 1] < totalPages - 1 ? (
              <PaginationItem>
                <PaginationEllipsis />
              </PaginationItem>
            ) : null}
            <PaginationItem>
              <Link
                to="/specs"
                search={linkSearch(totalPages)}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "icon" })
                )}
              >
                {totalPages}
              </Link>
            </PaginationItem>
          </>
        ) : null}

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
              to="/specs"
              search={linkSearch(current + 1)}
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

function pageWindow(
  current: number,
  total: number,
  size: number
): Array<number> {
  if (total <= 0) return []
  const half = Math.floor(size / 2)
  let start = Math.max(1, current - half)
  const end = Math.min(total, start + size - 1)
  start = Math.max(1, end - size + 1)
  const out: Array<number> = []
  for (let i = start; i <= end; i++) out.push(i)
  return out
}
