// depends-on-picker.tsx
//
// Multi-select dialog that edits a task's depends_on set. The parent task
// page mounts this when the user clicks the "+" icon next to "Depends on".
//
// Contract:
//   - `taskId` is the SUBJECT task being edited; it is always excluded from
//     the candidate list (a task cannot depend on itself).
//   - `current` is the existing set of blocker IDs; their checkboxes are
//     pre-selected on open and form the "before" state for the diff.
//   - On Save, the dialog calls `onConfirm(nextIds)` with the *complete*
//     replacement set. The parent is responsible for the API call and
//     re-rendering off the response — keeping this component network-free
//     means it can be reused for other multi-select-against-tasks flows.
//   - On Cancel/close, no callback fires; the parent's "current" prop is the
//     source of truth on the next open.
//
// Search filters client-side over id + title + summary so it is responsive
// even with hundreds of tasks; the parent prefetches the full task list and
// passes it in via `tasks`. If the project ever outgrows that, switch to a
// server-side query against /api/search.
import { useEffect, useMemo, useState } from "react"

import type { ListTaskItem } from "@/lib/api/tasks"
import { humanizeSlug } from "@/lib/format"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"

export function DependsOnPicker({
  open,
  onOpenChange,
  taskId,
  tasks,
  loading,
  current,
  onConfirm,
  saving,
}: {
  open: boolean
  onOpenChange: (next: boolean) => void
  taskId: string
  tasks: Array<ListTaskItem>
  loading: boolean
  current: Array<string>
  onConfirm: (ids: Array<string>) => void
  saving: boolean
}) {
  const [query, setQuery] = useState("")
  const [selected, setSelected] = useState<Set<string>>(() => new Set(current))

  // Reset local state every time the dialog opens so a previous "Cancel"
  // doesn't leak into the next open. Use the open transition (not a deep
  // equality check on `current`) to keep the effect cheap.
  useEffect(() => {
    if (open) {
      setSelected(new Set(current))
      setQuery("")
    }
  }, [open, current])

  // Candidate list: never include the subject task; apply a simple
  // case-insensitive substring filter across id/title/summary.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return tasks.filter((t) => {
      if (t.id === taskId) return false
      if (!q) return true
      return (
        t.id.toLowerCase().includes(q) ||
        t.title.toLowerCase().includes(q) ||
        t.summary.toLowerCase().includes(q)
      )
    })
  }, [tasks, taskId, query])

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function confirm() {
    // Order by id for a stable serialised representation; the server stores
    // dependencies as a set so order doesn't matter, but keeping it stable
    // makes test snapshots and markdown diffs predictable.
    const ids = Array.from(selected).sort()
    onConfirm(ids)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit dependencies</DialogTitle>
          <DialogDescription>
            Select the tasks that must be completed before {taskId}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter tasks…"
            aria-label="Filter tasks"
            autoFocus
          />
          <div
            role="listbox"
            aria-multiselectable
            aria-label="Candidate dependencies"
            className="max-h-72 overflow-y-auto rounded-md border"
          >
            {loading ? (
              <p className="p-4 text-sm text-muted-foreground">
                Loading tasks…
              </p>
            ) : filtered.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">
                {tasks.length === 0
                  ? "No tasks available."
                  : `No tasks match “${query}”.`}
              </p>
            ) : (
              <ul className="divide-y">
                {filtered.map((t) => {
                  const isChecked = selected.has(t.id)
                  return (
                    <li key={t.id}>
                      <label className="flex cursor-pointer items-start gap-3 px-3 py-2 hover:bg-accent/40">
                        <Checkbox
                          checked={isChecked}
                          onCheckedChange={() => toggle(t.id)}
                          aria-label={`Toggle ${t.id}`}
                          className="mt-1"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs text-muted-foreground">
                              {t.id}
                            </span>
                            <Badge variant="outline" className="text-[10px]">
                              {humanizeSlug(t.status)}
                            </Badge>
                          </div>
                          <p className="truncate text-sm font-medium">
                            {t.title}
                          </p>
                          {t.summary ? (
                            <p className="line-clamp-2 text-xs text-muted-foreground">
                              {t.summary}
                            </p>
                          ) : null}
                        </div>
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            {selected.size} selected
          </p>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button type="button" onClick={confirm} disabled={saving}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
