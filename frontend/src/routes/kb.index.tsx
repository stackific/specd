import { useEffect, useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import type { KBDocSummary } from "@/lib/api/kb"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { fetchKBList } from "@/lib/api/kb"
import { formatDateTime, formatRelativeTime } from "@/lib/format"

export const Route = createFileRoute("/kb/")({
  component: KBList,
})

function KBList() {
  const navigate = useNavigate()
  const [items, setItems] = useState<Array<KBDocSummary> | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    setError(null)
    fetchKBList(ctrl.signal)
      .then((res) => {
        setItems(res.items)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (ctrl.signal.aborted) return
        setError(err instanceof Error ? err.message : "Failed to load KB docs")
        setLoading(false)
      })
    return () => ctrl.abort()
  }, [reloadKey])

  const goTo = (id: string) => {
    navigate({ to: "/kb/$id", params: { id } })
  }

  return (
    <div className="container mx-auto max-w-5xl space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">
          Knowledge base
        </h1>
        <p className="text-muted-foreground">
          Searchable reference docs chunked for retrieval.
        </p>
      </header>

      {error ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <p className="text-sm text-destructive">{error}</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setReloadKey((k) => k + 1)}
            >
              Retry
            </Button>
          </CardContent>
        </Card>
      ) : loading ? (
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-32">ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead className="w-40">Source type</TableHead>
                <TableHead className="w-48">Added at</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-20" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-64" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-32" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : items && items.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            No KB docs yet.
          </CardContent>
        </Card>
      ) : (
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-32">ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead className="w-40">Source type</TableHead>
                <TableHead className="w-48">Added at</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items?.map((doc) => (
                <TableRow
                  key={doc.id}
                  role="link"
                  tabIndex={0}
                  className="cursor-pointer"
                  onClick={() => goTo(doc.id)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault()
                      goTo(doc.id)
                    }
                  }}
                >
                  <TableCell className="font-mono text-xs">{doc.id}</TableCell>
                  <TableCell className="font-medium">{doc.title}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {doc.source_type}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    <time
                      dateTime={doc.added_at}
                      title={formatDateTime(doc.added_at)}
                    >
                      {formatRelativeTime(doc.added_at)}
                    </time>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
