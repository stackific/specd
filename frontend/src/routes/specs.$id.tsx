import { useEffect, useMemo, useState } from "react"
import { Link, createFileRoute, useParams } from "@tanstack/react-router"
import { ListPlus, PlusIcon, Trash2 } from "lucide-react"
import { toast } from "sonner"
import type {
  GetSpecResponse,
  ListSpecItem,
  SpecRef,
  SpecTask,
} from "@/lib/api/specs"
import { getSpec, listSpecs, setLinkedSpecs, unlinkSpec } from "@/lib/api/specs"
import { ApiError } from "@/lib/api"
import { deleteTask } from "@/lib/api/tasks"
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
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Markdown } from "@/components/common/markdown"
import { CopyableHash } from "@/components/common/copyable-hash"
import { SpecLinkPicker } from "@/components/specs/spec-link-picker"
import { formatDateTime, formatRelativeTime, humanizeSlug } from "@/lib/format"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"

export const Route = createFileRoute("/specs/$id")({
  // No search params on the detail route; stripping prevents the list-page
  // filter (view/type/page/page_size) from leaking into the URL when a user
  // navigates from /specs to /specs/$id.
  validateSearch: () => ({}),
  component: SpecDetailPage,
})

function SpecDetailPage() {
  const { id } = useParams({ from: "/specs/$id" })
  const [data, setData] = useState<GetSpecResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<{
    status: number
    message: string
  } | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  // pendingDelete drives the AlertDialog: set it to open the confirm,
  // clear it to dismiss. Holding the full SpecTask (not just the ID) lets
  // the dialog show the title for context.
  const [pendingDelete, setPendingDelete] = useState<SpecTask | null>(null)
  const [deleting, setDeleting] = useState(false)
  // Same pattern for the "remove a linked-spec relationship" affordance.
  // Held separately so the two dialogs don't trip over each other and so
  // their disabled-during-flight flags stay independent.
  const [pendingUnlink, setPendingUnlink] = useState<SpecRef | null>(null)
  const [unlinking, setUnlinking] = useState(false)
  // Linked-specs picker state. Lazy-loaded on first open so the page stays
  // a single GET /api/specs/{id} until the user actually wants to edit
  // links. Mirrors the depends-on picker on routes/tasks.$id.tsx.
  const [linkPickerOpen, setLinkPickerOpen] = useState(false)
  const [linkPickerSpecs, setLinkPickerSpecs] = useState<Array<ListSpecItem>>(
    []
  )
  const [linkPickerLoading, setLinkPickerLoading] = useState(false)
  const [savingLinks, setSavingLinks] = useState(false)
  // Child-task add affordance is a placeholder until the re-parent endpoint
  // lands. Re-parenting requires moving TASK-N.md across spec dirs and
  // updating tasks.spec_id + tasks.path, which is its own scoped change.
  // The dialog is intentionally bare so users see the affordance and know
  // it's coming, without a fake-functional shell that misleads them.
  const [childAddOpen, setChildAddOpen] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError(null)
    getSpec(id, controller.signal)
      .then((res) => {
        setData(res)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        if (err instanceof ApiError) {
          setError({
            status: err.status,
            message: err.message || "Failed to load spec.",
          })
        } else {
          setError({ status: 0, message: "Failed to load spec." })
        }
        setLoading(false)
      })
    return () => controller.abort()
  }, [id, reloadKey])

  // openLinkPicker fetches the full spec list on first open. Subsequent
  // opens reuse the cached list — same lazy-load contract as the depends-on
  // picker on routes/tasks.$id.tsx.
  async function openLinkPicker() {
    setLinkPickerOpen(true)
    if (linkPickerSpecs.length > 0 || linkPickerLoading) return
    setLinkPickerLoading(true)
    try {
      const res = await listSpecs({ pageSize: 500 })
      setLinkPickerSpecs(res.items)
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to load specs"
      toast.error(msg)
    } finally {
      setLinkPickerLoading(false)
    }
  }

  // saveLinkedSpecs is the single write path for both the picker's "Save"
  // and any future inline action that mutates the link set. Pass the
  // COMPLETE next set; the server diffs against current rows and rewrites
  // the affected spec.md files. The fresh detail comes back so we drop it
  // straight into local state without a follow-up GET.
  async function saveLinkedSpecs(next: Array<string>) {
    if (savingLinks) return
    setSavingLinks(true)
    try {
      const updated = await setLinkedSpecs(id, next)
      setData(updated)
      setLinkPickerOpen(false)
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : "Failed to update linked specs"
      toast.error(msg)
    } finally {
      setSavingLinks(false)
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto max-w-4xl space-y-6 p-6">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  }

  if (error) {
    const notFound = error.status === 404
    return (
      <div className="container mx-auto max-w-3xl space-y-6 p-6">
        <Card>
          <CardHeader>
            <CardTitle>
              {notFound ? "Spec not found" : "Failed to load spec"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 text-sm text-muted-foreground">
            <p>
              {notFound ? `No spec exists with id "${id}".` : error.message}
            </p>
            <div className="flex gap-2">
              <Button asChild variant="outline" size="sm">
                <Link
                  to="/specs"
                  search={{
                    view: "grouped",
                    type: "all",
                    page: 1,
                    page_size: 20,
                  }}
                >
                  Back to specs
                </Link>
              </Button>
              {!notFound ? (
                <Button size="sm" onClick={() => setReloadKey((k) => k + 1)}>
                  Retry
                </Button>
              ) : null}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) return null

  return (
    <div className="container mx-auto max-w-4xl space-y-6 p-6">
      <div className="flex items-center justify-between gap-2">
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link
                  to="/specs"
                  search={{
                    view: "grouped",
                    type: "all",
                    page: 1,
                    page_size: 20,
                  }}
                >
                  Specs
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>{data.id}</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
        <Link
          to="/specs"
          search={{ view: "grouped", type: data.type, page: 1, page_size: 20 }}
          aria-label={`Filter specs by ${data.type}`}
        >
          <Badge
            variant="outline"
            className="capitalize transition-colors hover:bg-accent"
          >
            {data.type}
          </Badge>
        </Link>
      </div>

      <header className="space-y-3">
        <h1 className="text-3xl font-semibold tracking-tight">{data.title}</h1>
        {data.summary ? (
          <p className="text-base text-muted-foreground">{data.summary}</p>
        ) : null}
      </header>

      {data.body ? (
        <div className="text-sm [&_h1]:text-2xl [&_h2]:text-xl [&_h3]:text-lg [&_h4]:text-sm [&_li]:leading-6 [&_p]:leading-6">
          <Markdown>{stripAcceptanceCriteria(data.body)}</Markdown>
        </div>
      ) : null}

      {data.claims.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Acceptance criteria</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="list-disc space-y-1 pl-5 text-sm">
              {data.claims.map((claim) => (
                <li key={claim.position}>{claim.text}</li>
              ))}
            </ul>
          </CardContent>
        </Card>
      ) : null}

      <ChildTasks
        tasks={data.tasks}
        onRequestDelete={setPendingDelete}
        onAdd={() => setChildAddOpen(true)}
      />

      <Dialog open={childAddOpen} onOpenChange={setChildAddOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add child tasks</DialogTitle>
            <DialogDescription>
              This will let you attach existing tasks to {data.id}. We&apos;re
              wiring up the re-parent endpoint (move + rewrite the markdown file
              across spec directories) — it will be updated very soon.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" onClick={() => setChildAddOpen(false)}>
              Got it
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open && !deleting) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {pendingDelete?.id}?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDelete?.title
                ? `"${pendingDelete.title}" will be removed from the database and its
                   markdown file deleted from disk. This cannot be undone.`
                : "This task will be removed from the database and its markdown file deleted from disk. This cannot be undone."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={deleting}
              onClick={async (e) => {
                // Stop AlertDialog's default close-on-action so we can
                // await the network round-trip and only close on success.
                e.preventDefault()
                if (!pendingDelete) return
                setDeleting(true)
                try {
                  await deleteTask(pendingDelete.id)
                  setData((cur) =>
                    cur
                      ? {
                          ...cur,
                          tasks: cur.tasks.filter(
                            (t) => t.id !== pendingDelete.id
                          ),
                        }
                      : cur
                  )
                  toast.success(`Deleted ${pendingDelete.id}`)
                  setPendingDelete(null)
                } catch (err) {
                  const msg =
                    err instanceof ApiError
                      ? err.message
                      : err instanceof Error
                        ? err.message
                        : "Failed to delete task"
                  toast.error(msg)
                } finally {
                  setDeleting(false)
                }
              }}
            >
              {deleting ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <LinkedSpecs
        refs={data.linked_specs_refs}
        onRequestUnlink={setPendingUnlink}
        onAdd={openLinkPicker}
      />

      <SpecLinkPicker
        open={linkPickerOpen}
        onOpenChange={setLinkPickerOpen}
        specId={id}
        specs={linkPickerSpecs}
        loading={linkPickerLoading}
        current={data.linked_specs_refs.map((r) => r.id)}
        onConfirm={saveLinkedSpecs}
        saving={savingLinks}
      />

      <AlertDialog
        open={pendingUnlink !== null}
        onOpenChange={(open) => {
          if (!open && !unlinking) setPendingUnlink(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unlink {pendingUnlink?.id}?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingUnlink?.title
                ? `The relationship to "${pendingUnlink.title}" will be removed
                   from both specs. The specs themselves are not deleted; only
                   the link.`
                : "The relationship will be removed from both specs. Neither spec is deleted."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={unlinking}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={unlinking}
              onClick={async (e) => {
                e.preventDefault()
                if (!pendingUnlink) return
                setUnlinking(true)
                try {
                  const updated = await unlinkSpec(id, pendingUnlink.id)
                  setData(updated)
                  toast.success(`Unlinked ${pendingUnlink.id}`)
                  setPendingUnlink(null)
                } catch (err) {
                  const msg =
                    err instanceof ApiError
                      ? err.message
                      : err instanceof Error
                        ? err.message
                        : "Failed to unlink spec"
                  toast.error(msg)
                } finally {
                  setUnlinking(false)
                }
              }}
            >
              {unlinking ? "Unlinking…" : "Unlink"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <footer className="grid gap-x-6 gap-y-1 border-t pt-4 text-xs text-muted-foreground sm:grid-cols-2">
        {data.created_by ? (
          <p className="sm:col-start-1">Created by: {data.created_by}</p>
        ) : null}
        {data.updated_by ? (
          <p className="sm:col-start-1">Updated by: {data.updated_by}</p>
        ) : null}
        <p className="sm:col-start-2 sm:text-right">
          Created:{" "}
          <time
            dateTime={data.created_at}
            title={formatDateTime(data.created_at)}
          >
            {formatRelativeTime(data.created_at)}
          </time>
        </p>
        <p className="sm:col-start-2 sm:text-right">
          Updated:{" "}
          <time
            dateTime={data.updated_at}
            title={formatDateTime(data.updated_at)}
          >
            {formatRelativeTime(data.updated_at)}
          </time>
        </p>
        <CopyableHash
          hash={data.content_hash}
          className="sm:col-start-2 sm:justify-self-end"
        />
      </footer>
    </div>
  )
}

// ChildTasks renders the spec's tasks grouped by status, with a Delete
// button per row. Each row is a flex of (Link covering the meta + summary)
// + (Trash button). The Link and Button are siblings — never nest a Button
// inside a Link or the click events fight each other. The button delegates
// confirmation to the parent (via onRequestDelete) so the AlertDialog lives
// at the page level, not inside the list.
//
// `onAdd` opens the page-level "add child tasks" dialog. The card renders
// unconditionally so the affordance is reachable on a spec with no tasks
// yet — same shape as LinkedSpecs.
function ChildTasks({
  tasks,
  onRequestDelete,
  onAdd,
}: {
  tasks: Array<SpecTask>
  onRequestDelete: (task: SpecTask) => void
  onAdd: () => void
}) {
  const grouped = useMemo(() => {
    const map = new Map<string, Array<SpecTask>>()
    for (const t of tasks) {
      const key = t.status || "unknown"
      const arr = map.get(key) ?? []
      arr.push(t)
      map.set(key, arr)
    }
    return Array.from(map.entries())
  }, [tasks])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Child tasks</CardTitle>
        <CardAction>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-7"
            aria-label="Add child tasks"
            onClick={onAdd}
          >
            <ListPlus aria-hidden="true" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-4">
        {tasks.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No child tasks yet. Use <span aria-hidden="true">+</span>
            <span className="sr-only">the add button</span> to attach tasks to
            this spec.
          </p>
        ) : null}
        {grouped.map(([status, items]) => (
          <section
            key={status}
            aria-label={`${status} tasks`}
            className="space-y-2"
          >
            <div className="flex items-center gap-2">
              <Badge variant="outline">{humanizeSlug(status)}</Badge>
              <span className="text-xs text-muted-foreground">
                {items.length}
              </span>
            </div>
            <ul className="divide-y rounded-md border">
              {items.map((t) => (
                <li
                  key={t.id}
                  className="flex items-stretch hover:bg-accent/50"
                >
                  <Link
                    to="/tasks/$id"
                    params={{ id: t.id }}
                    search={{}}
                    className="flex flex-1 flex-col gap-1 px-3 py-2 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="font-mono text-xs text-muted-foreground">
                        {t.id}
                      </span>
                      <span className="truncate font-medium">{t.title}</span>
                    </div>
                    {t.summary ? (
                      <span className="truncate text-sm text-muted-foreground sm:max-w-md">
                        {t.summary}
                      </span>
                    ) : null}
                  </Link>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="mr-1 self-center text-muted-foreground hover:text-destructive"
                    onClick={(e) => {
                      e.stopPropagation()
                      onRequestDelete(t)
                    }}
                    aria-label={`Delete ${t.id}`}
                  >
                    <Trash2 aria-hidden="true" />
                  </Button>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </CardContent>
    </Card>
  )
}

// LinkedSpecs renders the spec's outbound links as the same row layout as
// ChildTasks: id (mono) + title + summary snippet, plus a Trash button
// that asks the parent to confirm and unlink. The "+" affordance lives in
// the CardAction slot so the shadcn header grid puts it hard against the
// right edge — same recipe as the depends-on header on tasks.$id.tsx.
//
// The card renders unconditionally so the "+" is reachable even when the
// spec has no links. Same Button-as-sibling-of-Link rule applies — never
// nest a Button inside a Link.
function LinkedSpecs({
  refs,
  onRequestUnlink,
  onAdd,
}: {
  refs: Array<SpecRef>
  onRequestUnlink: (ref: SpecRef) => void
  onAdd: () => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Linked specs</CardTitle>
        <CardAction>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-7"
            aria-label="Add linked specs"
            onClick={onAdd}
          >
            <PlusIcon aria-hidden="true" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {refs.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No linked specs. Use <span aria-hidden="true">+</span>
            <span className="sr-only">the add button</span> to pick related
            specs.
          </p>
        ) : (
          <ul className="divide-y rounded-md border">
            {refs.map((ref) => (
              <li
                key={ref.id}
                className="flex items-stretch hover:bg-accent/50"
              >
                <Link
                  to="/specs/$id"
                  params={{ id: ref.id }}
                  search={{}}
                  className="flex flex-1 flex-col gap-1 px-3 py-2 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="font-mono text-xs text-muted-foreground">
                      {ref.id}
                    </span>
                    <span className="truncate font-medium">
                      {ref.title || "(missing title)"}
                    </span>
                  </div>
                  {ref.summary ? (
                    <span className="truncate text-sm text-muted-foreground sm:max-w-md">
                      {ref.summary}
                    </span>
                  ) : null}
                </Link>
                <Button
                  variant="ghost"
                  size="icon"
                  className="mr-1 self-center text-muted-foreground hover:text-destructive"
                  onClick={(e) => {
                    e.stopPropagation()
                    onRequestUnlink(ref)
                  }}
                  aria-label={`Unlink ${ref.id}`}
                >
                  <Trash2 aria-hidden="true" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

// Strips a top-level "## Acceptance criteria" section (and its body, up to
// the next ## heading or end of document) from the spec body markdown.
// Spec bodies redundantly include the acceptance criteria as a markdown
// list; the structured `claims` are already rendered in their own card.
//
// Note: no `m` flag — we want `$` to mean end-of-string, not end-of-line,
// so the lazy match continues past the heading to swallow the bullets.
// `(?:\n|^)` anchors the match at a line boundary instead.
function stripAcceptanceCriteria(md: string): string {
  const re = /(?:\n|^)##\s+Acceptance\s+Criteria\b[\s\S]*?(?=\n##\s|$)/i
  return md
    .replace(re, "")
    .replace(/\n{3,}/g, "\n\n")
    .trim()
}
