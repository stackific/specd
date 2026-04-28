import { useEffect, useState } from "react"
import { Link, createFileRoute, useParams } from "@tanstack/react-router"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import rehypeSanitize from "rehype-sanitize"
import type { KBDetailResponse } from "@/lib/api/kb"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { ApiError } from "@/lib/api"
import { fetchKBDetail } from "@/lib/api/kb"
import { formatDateTime, formatRelativeTime } from "@/lib/format"

export const Route = createFileRoute("/kb/$id")({
  component: KBDetail,
})

function KBDetail() {
  const { id } = useParams({ from: "/kb/$id" })
  const [data, setData] = useState<KBDetailResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    setError(null)
    setNotFound(false)
    fetchKBDetail(id, ctrl.signal)
      .then((res) => {
        setData(res)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (ctrl.signal.aborted) return
        if (err instanceof ApiError && err.status === 404) {
          setNotFound(true)
        } else {
          setError(err instanceof Error ? err.message : "Failed to load KB doc")
        }
        setLoading(false)
      })
    return () => ctrl.abort()
  }, [id])

  if (loading) {
    return (
      <div className="container mx-auto max-w-4xl space-y-6 p-6">
        <Skeleton className="h-9 w-2/3" />
        <Skeleton className="h-5 w-full" />
        <Skeleton className="h-4 w-1/2" />
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-1/2" />
              </CardHeader>
              <CardContent className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  if (notFound) {
    return (
      <div className="container mx-auto max-w-4xl p-6">
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <p className="text-sm">KB doc not found</p>
            <Button asChild variant="outline" size="sm">
              <Link to="/kb">Back to knowledge base</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container mx-auto max-w-4xl p-6">
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <p className="text-sm text-destructive">{error}</p>
            <Button asChild variant="outline" size="sm">
              <Link to="/kb">Back to knowledge base</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) return null

  const { Doc, Chunks } = data
  const chunks = (Chunks ?? []).slice().sort((a, b) => a.Position - b.Position)

  return (
    <div className="container mx-auto max-w-4xl space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">{Doc.Title}</h1>
        {Doc.Summary ? (
          <p className="text-muted-foreground">{Doc.Summary}</p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          <span className="font-mono">{Doc.ID}</span>
          {" • "}
          {Doc.SourceType}
          {Doc.AddedBy ? ` • added by ${Doc.AddedBy}` : ""}
          {Doc.AddedAt ? (
            <>
              {" • "}
              <time dateTime={Doc.AddedAt} title={formatDateTime(Doc.AddedAt)}>
                added {formatRelativeTime(Doc.AddedAt)}
              </time>
            </>
          ) : null}
          {Doc.Path ? ` • ${Doc.Path}` : ""}
        </p>
      </header>

      <div className="space-y-4">
        {chunks.map((chunk) => (
          <Card key={chunk.Position}>
            <CardHeader>
              <CardTitle className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Chunk {chunk.Position}
              </CardTitle>
              {chunk.Summary ? (
                <CardDescription>{chunk.Summary}</CardDescription>
              ) : null}
            </CardHeader>
            <CardContent>
              <div className="prose dark:prose-invert max-w-none">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeSanitize]}
                >
                  {chunk.Text}
                </ReactMarkdown>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
