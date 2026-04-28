import { useEffect, useState } from "react"
import { Link, createFileRoute, useParams } from "@tanstack/react-router"
import { PlusIcon, Trash2 } from "lucide-react"
import { toast } from "sonner"
import type {
  ListTaskItem,
  TaskDetailResponse,
  TaskRef,
  TaskResponse,
} from "@/lib/api/tasks"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import { CriteriaList } from "@/components/tasks/criteria-list"
import { DependsOnPicker } from "@/components/tasks/depends-on-picker"
import { CopyableHash } from "@/components/common/copyable-hash"
import { Markdown } from "@/components/common/markdown"
import { formatDateTime, formatRelativeTime } from "@/lib/format"
import {
  fetchTaskDetail,
  listTasks,
  setTaskDependsOn,
  toggleCriterion,
} from "@/lib/api/tasks"

export const Route = createFileRoute("/tasks/$id")({
  component: TaskDetail,
})

function TaskDetail() {
  const { id } = useParams({ from: "/tasks/$id" })
  const [data, setData] = useState<TaskDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState<Set<number>>(new Set())

  // Depends-on editor state. We lazy-load the candidate task list the first
  // time the picker opens so the page itself stays a single GET on
  // /api/tasks/{id}. `pendingRemove` drives the confirm AlertDialog for the
  // inline trash button — same pattern as the Delete-task flow on
  // routes/specs.$id.tsx.
  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickerTasks, setPickerTasks] = useState<Array<ListTaskItem>>([])
  const [pickerLoading, setPickerLoading] = useState(false)
  const [savingDeps, setSavingDeps] = useState(false)
  const [pendingRemove, setPendingRemove] = useState<TaskRef | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    fetchTaskDetail(id)
      .then((d) => {
        if (!cancelled) setData(d)
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  // openPicker fetches the full task list on first open. Subsequent opens
  // reuse the cached list — when an edit invalidates it (status change, new
  // task created elsewhere), bumping `pickerLoading=true` here would force a
  // refetch; for the MVP we accept stale candidates between detail loads.
  async function openPicker() {
    setPickerOpen(true)
    if (pickerTasks.length > 0 || pickerLoading) return
    setPickerLoading(true)
    try {
      const res = await listTasks({ pageSize: 500 })
      setPickerTasks(res.items)
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to load tasks"
      toast.error(msg)
    } finally {
      setPickerLoading(false)
    }
  }

  // saveDependsOn is the single write path for both the picker's "Save" and
  // the inline X-icon "remove". Pass the COMPLETE next set; the server
  // replaces the row set in one transaction and returns the updated detail
  // payload, which we drop straight into local state.
  async function saveDependsOn(next: Array<string>) {
    if (savingDeps) return
    setSavingDeps(true)
    try {
      const updated = await setTaskDependsOn(id, next)
      setData(updated)
      setPickerOpen(false)
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : "Failed to update dependencies"
      toast.error(msg)
    } finally {
      setSavingDeps(false)
    }
  }

  const handleToggle = async (position: number) => {
    if (!data) return
    const previous = data
    // Optimistic toggle.
    const optimisticTask: TaskResponse = {
      ...data.task,
      criteria: data.task.criteria.map((c) =>
        c.position === position ? { ...c, checked: c.checked === 1 ? 0 : 1 } : c
      ),
    }
    const completed = optimisticTask.criteria.filter(
      (c) => c.checked === 1
    ).length
    setData({ ...data, task: optimisticTask, completed_count: completed })
    setPending((s) => {
      const next = new Set(s)
      next.add(position)
      return next
    })
    try {
      const updated = await toggleCriterion(id, position)
      setData((cur) =>
        cur
          ? {
              ...cur,
              task: updated,
              completed_count: updated.criteria.filter((c) => c.checked === 1)
                .length,
              total_criteria: updated.criteria.length,
            }
          : cur
      )
    } catch (err) {
      setData(previous)
      const msg =
        err instanceof Error ? err.message : "Failed to toggle criterion"
      toast.error(msg)
    } finally {
      setPending((s) => {
        const next = new Set(s)
        next.delete(position)
        return next
      })
    }
  }

  if (loading) {
    return <TaskDetailSkeleton />
  }
  if (error) {
    return (
      <div className="container mx-auto max-w-3xl space-y-4 p-6">
        <h1 className="text-2xl font-semibold tracking-tight">{id}</h1>
        <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-6">
          <p className="text-sm font-medium text-destructive">
            Failed to load task
          </p>
          <p className="mt-1 text-xs text-muted-foreground">{error}</p>
        </div>
      </div>
    )
  }
  if (!data) return null

  const {
    task,
    parent_spec,
    status_label,
    body_clean,
    completed_count,
    total_criteria,
    depends_on_refs,
    linked_task_refs,
  } = data

  return (
    <div className="container mx-auto max-w-4xl space-y-6 p-6">
      <header className="space-y-3">
        {parent_spec ? (
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span>Parent spec</span>
            <Link
              to="/specs/$id"
              params={{ id: parent_spec.id }}
              search={{}}
              className="inline-flex"
            >
              <Badge variant="outline" className="hover:bg-muted">
                {parent_spec.id} — {parent_spec.title}
              </Badge>
            </Link>
          </div>
        ) : null}
        <div className="flex flex-wrap items-start gap-3">
          <h1 className="flex-1 text-3xl font-semibold tracking-tight">
            {task.title}
          </h1>
          <Badge variant="secondary">{status_label}</Badge>
        </div>
        <p className="font-mono text-xs text-muted-foreground">{task.id}</p>
        {task.summary ? (
          <p className="text-base text-muted-foreground">{task.summary}</p>
        ) : null}
      </header>

      <Separator />

      {body_clean.trim() ? (
        <section aria-labelledby="body-heading" className="space-y-2">
          <h2 id="body-heading" className="text-lg font-semibold">
            Description
          </h2>
          <div className="text-sm [&_h1]:text-2xl [&_h2]:text-xl [&_h3]:text-lg [&_h4]:text-sm [&_li]:leading-6 [&_p]:leading-6">
            <Markdown>{body_clean}</Markdown>
          </div>
        </section>
      ) : null}

      <section className="space-y-3" aria-labelledby="criteria-heading">
        <div className="flex items-baseline justify-between">
          <h2 id="criteria-heading" className="text-lg font-semibold">
            Acceptance criteria
          </h2>
          {total_criteria > 0 ? (
            <span className="text-xs text-muted-foreground">
              {completed_count} / {total_criteria} complete
            </span>
          ) : null}
        </div>
        <CriteriaList
          criteria={task.criteria}
          pendingPositions={pending}
          onToggle={handleToggle}
        />
      </section>

      {linked_task_refs.length > 0 ? (
        <section className="space-y-2" aria-labelledby="linked-heading">
          <h2 id="linked-heading" className="text-lg font-semibold">
            Linked tasks
          </h2>
          <ul role="list" className="flex flex-wrap gap-2">
            {linked_task_refs.map((ref) => (
              <li key={ref.id} role="listitem">
                <Link to="/tasks/$id" params={{ id: ref.id }} search={{}}>
                  <Badge variant="outline" className="hover:bg-muted">
                    {ref.id}
                    {ref.title ? ` — ${ref.title}` : ""}
                  </Badge>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Depends on</CardTitle>
          <CardAction>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-7"
              aria-label="Add dependencies"
              onClick={openPicker}
            >
              <PlusIcon aria-hidden="true" />
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {depends_on_refs.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No dependencies. Use <span aria-hidden="true">+</span>
              <span className="sr-only">the add button</span> to pick blocker
              tasks.
            </p>
          ) : (
            <ul role="list" className="divide-y rounded-md border">
              {depends_on_refs.map((ref) => (
                <li
                  key={ref.id}
                  role="listitem"
                  className="flex items-start hover:bg-accent/50"
                >
                  <Link
                    to="/tasks/$id"
                    params={{ id: ref.id }}
                    search={{}}
                    className="flex min-w-0 flex-1 items-start gap-3 px-3 py-2"
                  >
                    <span
                      aria-hidden="true"
                      className="mt-2 inline-block size-1.5 shrink-0 rounded-full bg-amber-500"
                    />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-baseline sm:gap-3">
                      <div className="flex items-baseline gap-3">
                        <span className="font-mono text-xs text-muted-foreground">
                          {ref.id}
                        </span>
                        <span className="truncate font-medium">
                          {ref.title}
                        </span>
                      </div>
                      {ref.summary ? (
                        <span className="line-clamp-1 min-w-0 text-xs text-muted-foreground sm:ml-auto sm:text-right">
                          {ref.summary}
                        </span>
                      ) : null}
                    </div>
                  </Link>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="mt-1 mr-1 size-7 shrink-0 text-muted-foreground hover:text-destructive"
                    aria-label={`Remove dependency ${ref.id}`}
                    disabled={savingDeps}
                    onClick={() => setPendingRemove(ref)}
                  >
                    <Trash2 aria-hidden="true" />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <DependsOnPicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        taskId={id}
        tasks={pickerTasks}
        loading={pickerLoading}
        current={depends_on_refs.map((r) => r.id)}
        onConfirm={saveDependsOn}
        saving={savingDeps}
      />

      <AlertDialog
        open={pendingRemove !== null}
        onOpenChange={(open) => {
          if (!open && !savingDeps) setPendingRemove(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Remove dependency on {pendingRemove?.id}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingRemove?.title
                ? `${id} will no longer be blocked by "${pendingRemove.title}". The task itself is not deleted.`
                : `${id} will no longer be blocked by ${pendingRemove?.id ?? "this task"}. The task itself is not deleted.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={savingDeps}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={savingDeps}
              onClick={async (e) => {
                // Mirror the Delete-task pattern: stop the dialog's default
                // close-on-action so we can await the network round-trip and
                // only close on success.
                e.preventDefault()
                if (!pendingRemove) return
                await saveDependsOn(
                  depends_on_refs
                    .filter((r) => r.id !== pendingRemove.id)
                    .map((r) => r.id)
                )
                setPendingRemove(null)
              }}
            >
              {savingDeps ? "Removing…" : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Separator />

      <footer className="grid gap-x-6 gap-y-1 text-xs text-muted-foreground sm:grid-cols-2">
        {task.created_by ? (
          <p className="sm:col-start-1">Created by: {task.created_by}</p>
        ) : null}
        {task.updated_by ? (
          <p className="sm:col-start-1">Updated by: {task.updated_by}</p>
        ) : null}
        <p className="sm:col-start-2 sm:text-right">
          Created:{" "}
          <time
            dateTime={task.created_at}
            title={formatDateTime(task.created_at)}
          >
            {formatRelativeTime(task.created_at)}
          </time>
        </p>
        <p className="sm:col-start-2 sm:text-right">
          Updated:{" "}
          <time
            dateTime={task.updated_at}
            title={formatDateTime(task.updated_at)}
          >
            {formatRelativeTime(task.updated_at)}
          </time>
        </p>
        <CopyableHash
          hash={task.content_hash}
          className="sm:col-start-2 sm:justify-self-end"
        />
      </footer>
    </div>
  )
}

function TaskDetailSkeleton() {
  return (
    <div className="container mx-auto max-w-4xl space-y-6 p-6">
      <Skeleton className="h-8 w-2/3" />
      <Skeleton className="h-4 w-1/2" />
      <Separator />
      <div className="space-y-2">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
      <Skeleton className="h-32 w-full" />
    </div>
  )
}
