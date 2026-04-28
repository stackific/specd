import { useDroppable } from "@dnd-kit/core"
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable"
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react"
import type { BoardColumn } from "@/lib/api/tasks"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { KanbanCard } from "@/components/tasks/kanban-card"

interface KanbanColumnProps {
  column: BoardColumn
  collapsed: boolean
  onToggleCollapse: (status: string) => void
}

export function KanbanColumn({
  column,
  collapsed,
  onToggleCollapse,
}: KanbanColumnProps) {
  const { setNodeRef, isOver } = useDroppable({
    id: `column-${column.status}`,
    data: { type: "column", status: column.status },
  })
  const cardIds = column.tasks.map((t) => t.id)

  if (collapsed) {
    return (
      <section
        className="flex h-full w-10 shrink-0 flex-col items-center gap-2 rounded-lg border bg-muted/30 py-3"
        aria-label={`${column.label} column (collapsed)`}
      >
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7"
          aria-label={`Expand ${column.label}`}
          aria-pressed="true"
          onClick={() => onToggleCollapse(column.status)}
        >
          <ChevronRightIcon className="size-4" />
        </Button>
        <Badge variant="secondary" className="text-[10px]">
          {column.tasks.length}
        </Badge>
        <div className="flex-1" aria-hidden="true">
          <span
            className="block text-xs font-medium tracking-wide whitespace-nowrap text-muted-foreground [writing-mode:vertical-rl]"
            style={{ transform: "rotate(180deg)" }}
          >
            {column.label}
          </span>
        </div>
      </section>
    )
  }

  return (
    <section
      className="flex w-full shrink-0 flex-col rounded-lg border bg-muted/30 md:w-auto md:min-w-[240px] md:flex-1 md:basis-0"
      aria-label={`${column.label} column`}
    >
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex min-w-0 items-center gap-2">
          <h2 className="truncate text-sm font-semibold">{column.label}</h2>
          <Badge variant="secondary" className="text-[10px]">
            {column.tasks.length}
          </Badge>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7"
          aria-label={`Collapse ${column.label}`}
          aria-pressed="false"
          onClick={() => onToggleCollapse(column.status)}
        >
          <ChevronLeftIcon className="size-4" />
        </Button>
      </header>
      <div
        ref={setNodeRef}
        className={cn(
          "flex flex-1 flex-col gap-2 p-2 transition-colors",
          isOver && "bg-accent/40"
        )}
      >
        <SortableContext items={cardIds} strategy={verticalListSortingStrategy}>
          {column.tasks.length === 0 ? (
            <p className="px-2 py-6 text-center text-xs text-muted-foreground">
              No tasks
            </p>
          ) : (
            column.tasks.map((card) => (
              <KanbanCard key={card.id} card={card} status={column.status} />
            ))
          )}
        </SortableContext>
      </div>
    </section>
  )
}
