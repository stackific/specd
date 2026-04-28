import type { TaskCriterion } from "@/lib/api/tasks"
import { Checkbox } from "@/components/ui/checkbox"

interface CriteriaListProps {
  criteria: Array<TaskCriterion>
  pendingPositions: Set<number>
  onToggle: (position: number) => void
}

export function CriteriaList({
  criteria,
  pendingPositions,
  onToggle,
}: CriteriaListProps) {
  if (criteria.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No acceptance criteria yet.
      </p>
    )
  }
  return (
    <ul role="list" className="space-y-2">
      {criteria.map((c) => {
        const id = `criterion-${c.position}`
        const isChecked = c.checked === 1
        const isPending = pendingPositions.has(c.position)
        return (
          <li
            role="listitem"
            key={c.position}
            className="flex items-start gap-3 rounded-md border bg-card p-3"
          >
            <Checkbox
              id={id}
              checked={isChecked}
              disabled={isPending}
              aria-pressed={isChecked}
              onCheckedChange={() => onToggle(c.position)}
              className="mt-0.5"
            />
            <label
              htmlFor={id}
              className={
                "text-sm leading-relaxed " +
                (isChecked ? "text-muted-foreground line-through" : "")
              }
            >
              {c.text}
            </label>
          </li>
        )
      })}
    </ul>
  )
}
