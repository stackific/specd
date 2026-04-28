// spec-link-picker.tsx
//
// Multi-select dialog that edits a spec's linked_specs set. Mirror of
// `tasks/depends-on-picker.tsx` for related-specs editing — same contract
// so the UX is consistent across the two relationship types:
//
//   - `specId` is the SUBJECT spec being edited; always excluded from the
//     candidate list (a spec cannot link to itself).
//   - `current` is the existing set of linked spec IDs; their checkboxes
//     are pre-selected on open.
//   - On Save, the dialog calls `onConfirm(nextIds)` with the *complete*
//     replacement set; the parent owns the API call and re-renders off
//     the response. Keeping this component network-free means it can also
//     be used for any future "pick from existing specs" flow.
//   - Filter is client-side over id/title/summary so it stays responsive
//     even with hundreds of specs; the parent prefetches the full list and
//     passes it via `specs`.
import { useEffect, useMemo, useState } from "react"

import type { ListSpecItem } from "@/lib/api/specs"
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

export function SpecLinkPicker({
  open,
  onOpenChange,
  specId,
  specs,
  loading,
  current,
  onConfirm,
  saving,
}: {
  open: boolean
  onOpenChange: (next: boolean) => void
  specId: string
  specs: Array<ListSpecItem>
  loading: boolean
  current: Array<string>
  onConfirm: (ids: Array<string>) => void
  saving: boolean
}) {
  const [query, setQuery] = useState("")
  const [selected, setSelected] = useState<Set<string>>(() => new Set(current))

  // Reset local state on every open so a previous "Cancel" doesn't leak.
  useEffect(() => {
    if (open) {
      setSelected(new Set(current))
      setQuery("")
    }
  }, [open, current])

  // Candidate list: never include the subject; case-insensitive substring
  // filter across id/title/summary.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return specs.filter((s) => {
      if (s.id === specId) return false
      if (!q) return true
      return (
        s.id.toLowerCase().includes(q) ||
        s.title.toLowerCase().includes(q) ||
        s.summary.toLowerCase().includes(q)
      )
    })
  }, [specs, specId, query])

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function confirm() {
    // Stable order keeps server-side log lines / markdown diffs predictable;
    // the join table itself is unordered.
    onConfirm(Array.from(selected).sort())
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit linked specs</DialogTitle>
          <DialogDescription>
            Select the specs related to {specId}. Links are bidirectional.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter specs…"
            aria-label="Filter specs"
            autoFocus
          />
          <div
            role="listbox"
            aria-multiselectable
            aria-label="Candidate links"
            className="max-h-72 overflow-y-auto rounded-md border"
          >
            {loading ? (
              <p className="p-4 text-sm text-muted-foreground">
                Loading specs…
              </p>
            ) : filtered.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">
                {specs.length === 0
                  ? "No specs available."
                  : `No specs match “${query}”.`}
              </p>
            ) : (
              <ul className="divide-y">
                {filtered.map((s) => {
                  const isChecked = selected.has(s.id)
                  return (
                    <li key={s.id}>
                      <label className="flex cursor-pointer items-start gap-3 px-3 py-2 hover:bg-accent/40">
                        <Checkbox
                          checked={isChecked}
                          onCheckedChange={() => toggle(s.id)}
                          aria-label={`Toggle ${s.id}`}
                          className="mt-1"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs text-muted-foreground">
                              {s.id}
                            </span>
                            <Badge
                              variant="outline"
                              className="text-[10px] capitalize"
                            >
                              {humanizeSlug(s.type)}
                            </Badge>
                          </div>
                          <p className="truncate text-sm font-medium">
                            {s.title}
                          </p>
                          {s.summary ? (
                            <p className="line-clamp-2 text-xs text-muted-foreground">
                              {s.summary}
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
