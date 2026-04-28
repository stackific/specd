import { useSortable } from "@dnd-kit/sortable"
import { Link } from "@tanstack/react-router"
import { GripVerticalIcon } from "lucide-react"
import type { BoardCard } from "@/lib/api/tasks"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface KanbanCardProps {
  card: BoardCard
  status: string
  // Optional. When the card has criteria progress to show.
  completed?: number
  total?: number
}

export function KanbanCard({ card, status }: KanbanCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: card.id,
    data: { type: "card", status, card },
  })

  const style: React.CSSProperties = {
    transform: transform
      ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0)${
          transform.scaleX !== 1 || transform.scaleY !== 1
            ? ` scaleX(${transform.scaleX}) scaleY(${transform.scaleY})`
            : ""
        }`
      : undefined,
    transition,
    opacity: isDragging ? 0.5 : undefined,
  }

  return (
    <Card
      ref={setNodeRef}
      style={style}
      className={cn(
        "group relative gap-2 py-3",
        isDragging && "ring-2 ring-ring"
      )}
    >
      <CardContent className="flex items-start gap-2 px-3">
        <button
          type="button"
          className="mt-0.5 cursor-grab touch-none text-muted-foreground hover:text-foreground active:cursor-grabbing"
          aria-label={`Drag ${card.id}`}
          {...attributes}
          {...listeners}
        >
          <GripVerticalIcon className="size-4" />
        </button>
        <div className="min-w-0 flex-1 space-y-1">
          <h3 className="text-sm leading-snug font-medium">
            <Link
              to="/tasks/$id"
              params={{ id: card.id }}
              search={{}}
              className="hover:underline"
            >
              {card.title}
            </Link>
          </h3>
          {card.summary ? (
            <p className="line-clamp-2 text-xs text-muted-foreground">
              {card.summary}
            </p>
          ) : null}
          <div className="flex flex-wrap items-center gap-1.5 pt-1">
            <Badge variant="outline" className="text-[10px]">
              {card.id}
            </Badge>
            {card.spec_id ? (
              <Badge
                variant="ghost"
                className="text-[10px] text-muted-foreground"
              >
                {card.spec_id}
              </Badge>
            ) : null}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
