import { useEffect, useState } from "react"
import { Link, createFileRoute } from "@tanstack/react-router"
import type { StatsResponse } from "@/lib/api/stats"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { getStats } from "@/lib/api/stats"

export const Route = createFileRoute("/welcome")({
  component: Welcome,
})

function Welcome() {
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    getStats(ctrl.signal)
      .then(setStats)
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        setError(err instanceof Error ? err.message : "Failed to load stats")
      })
    return () => ctrl.abort()
  }, [])

  const taskPct =
    stats && stats.tasks_total > 0
      ? Math.round((stats.tasks_done / stats.tasks_total) * 100)
      : 0

  return (
    <div className="container mx-auto max-w-5xl space-y-10 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">
          Welcome to specd
        </h1>
        <p className="text-muted-foreground">
          Spec-driven development — manage specs, tasks, and knowledge from one
          place.
        </p>
      </header>

      <section aria-labelledby="overview-heading" className="space-y-3">
        <h2
          id="overview-heading"
          className="text-xs font-medium tracking-wide text-muted-foreground uppercase"
        >
          Overview
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardDescription>Tasks completed</CardDescription>
              <CardTitle className="text-3xl font-semibold tabular-nums">
                {stats ? (
                  <>
                    {stats.tasks_done}
                    <span className="text-muted-foreground">
                      {" "}
                      / {stats.tasks_total}
                    </span>
                  </>
                ) : (
                  <Skeleton className="h-9 w-24" />
                )}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {stats ? (
                <>
                  <Progress
                    value={taskPct}
                    aria-label={`${stats.tasks_done} of ${stats.tasks_total} tasks completed`}
                  />
                  <p className="mt-2 text-xs text-muted-foreground">
                    {error ? error : `${taskPct}% done`}
                  </p>
                </>
              ) : (
                <Skeleton className="h-2 w-full" />
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardDescription>Specs</CardDescription>
              <CardTitle className="text-3xl font-semibold tabular-nums">
                {stats ? stats.specs : <Skeleton className="h-9 w-16" />}
              </CardTitle>
            </CardHeader>
            <CardContent className="text-xs text-muted-foreground">
              Across all spec types in this project.
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardDescription>Knowledge base</CardDescription>
              <CardTitle className="text-3xl font-semibold tabular-nums">
                {stats ? stats.kb_docs : <Skeleton className="h-9 w-16" />}
              </CardTitle>
            </CardHeader>
            <CardContent className="text-xs text-muted-foreground">
              Documents indexed and searchable.
            </CardContent>
          </Card>
        </div>
      </section>

      <section aria-labelledby="quicklinks-heading" className="space-y-3">
        <h2
          id="quicklinks-heading"
          className="text-xs font-medium tracking-wide text-muted-foreground uppercase"
        >
          Quick links
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Link to="/specs" className="block h-full">
            <Card className="h-full transition-colors hover:bg-accent">
              <CardHeader>
                <CardTitle>Specs</CardTitle>
                <CardDescription>
                  Capture business, functional, and non-functional requirements.
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Use the sidebar to browse, filter by type, or search across the
                project.
              </CardContent>
            </Card>
          </Link>
          <Link to="/tasks" className="block h-full">
            <Card className="h-full transition-colors hover:bg-accent">
              <CardHeader>
                <CardTitle>Tasks</CardTitle>
                <CardDescription>
                  Track work on a kanban with drag-and-drop.
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Each task lives alongside its spec on disk and stays in sync
                with the file.
              </CardContent>
            </Card>
          </Link>
          <Link to="/kb" className="block h-full">
            <Card className="h-full transition-colors hover:bg-accent">
              <CardHeader>
                <CardTitle>Knowledge base</CardTitle>
                <CardDescription>
                  Searchable reference docs chunked for retrieval.
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Hybrid BM25 + trigram search ranks results across specs, tasks,
                and KB docs.
              </CardContent>
            </Card>
          </Link>
          <Link to="/search" className="block h-full">
            <Card className="h-full transition-colors hover:bg-accent">
              <CardHeader>
                <CardTitle>Search</CardTitle>
                <CardDescription>
                  Find anything across specs, tasks, and the knowledge base.
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Press{" "}
                <kbd className="rounded border bg-muted px-1 font-mono text-xs">
                  ⌘K
                </kbd>{" "}
                from any page, or open the search route directly.
              </CardContent>
            </Card>
          </Link>
          <Link to="/docs" className="block h-full">
            <Card className="h-full transition-colors hover:bg-accent">
              <CardHeader>
                <CardTitle>Documentation</CardTitle>
                <CardDescription>
                  Tutorial, user guide, and CLI/API reference.
                </CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Start with the tutorial to walk through the spec-driven workflow
                end to end.
              </CardContent>
            </Card>
          </Link>
        </div>
      </section>
    </div>
  )
}
