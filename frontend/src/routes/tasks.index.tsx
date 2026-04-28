import { useEffect, useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCorners,
  useSensor,
  useSensors,
} from "@dnd-kit/core"
import { sortableKeyboardCoordinates } from "@dnd-kit/sortable"
import { toast } from "sonner"
import type { DragEndEvent } from "@dnd-kit/core"
import type { BoardFilter, BoardResponse } from "@/lib/api/tasks"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { KanbanColumn } from "@/components/tasks/kanban-column"
import { fetchBoard, moveTask } from "@/lib/api/tasks"

const COLLAPSED_KEY = "specd-kanban-collapsed"

interface TaskSearch {
  filter?: BoardFilter
}

export const Route = createFileRoute("/tasks/")({
  validateSearch: (raw: Record<string, unknown>): TaskSearch => {
    const f = raw.filter
    if (f === "incomplete" || f === "all") return { filter: f }
    return {}
  },
  component: TasksPage,
})

function readCollapsed(): Set<string> {
  if (typeof window === "undefined") return new Set()
  try {
    const raw = window.localStorage.getItem(COLLAPSED_KEY)
    if (!raw) return new Set()
    return new Set(
      raw
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean)
    )
  } catch {
    return new Set()
  }
}

function writeCollapsed(set: Set<string>) {
  try {
    window.localStorage.setItem(COLLAPSED_KEY, Array.from(set).join(","))
  } catch {
    /* noop */
  }
}

function TasksPage() {
  const search = Route.useSearch()
  const filter: BoardFilter = search.filter ?? "all"
  const navigate = useNavigate({ from: "/tasks" })

  const [board, setBoard] = useState<BoardResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [collapsed, setCollapsed] = useState<Set<string>>(() => readCollapsed())

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    fetchBoard(filter)
      .then((data) => {
        if (!cancelled) setBoard(data)
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
  }, [filter])

  const handleToggleCollapse = (status: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(status)) next.delete(status)
      else next.add(status)
      writeCollapsed(next)
      return next
    })
  }

  const handleFilterChange = (value: string) => {
    const next: BoardFilter = value === "incomplete" ? "incomplete" : "all"
    void navigate({ search: { filter: next } })
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    if (!board) return
    const { active, over } = event
    if (!over) return

    const activeId = String(active.id)
    const overId = String(over.id)

    // Find current location of the active card.
    const fromCol = board.columns.find((c) =>
      c.tasks.some((t) => t.id === activeId)
    )
    if (!fromCol) return

    // Determine destination column + position.
    let toStatus: string
    let toIndex: number

    const overData = over.data.current as
      | { type?: string; status?: string }
      | undefined
    if (overData?.type === "column") {
      toStatus = overData.status as string
      const toCol = board.columns.find((c) => c.status === toStatus)
      toIndex = toCol ? toCol.tasks.length : 0
    } else {
      // Hovering another card.
      const toCol = board.columns.find((c) =>
        c.tasks.some((t) => t.id === overId)
      )
      if (!toCol) return
      toStatus = toCol.status
      toIndex = toCol.tasks.findIndex((t) => t.id === overId)
      if (toIndex < 0) toIndex = toCol.tasks.length
    }

    // No-op?
    if (
      fromCol.status === toStatus &&
      fromCol.tasks.findIndex((t) => t.id === activeId) === toIndex
    ) {
      return
    }

    // Optimistic update.
    const previous = board
    const next: BoardResponse = {
      ...board,
      columns: board.columns.map((c) => ({ ...c, tasks: [...c.tasks] })),
    }
    const nextFromCol = next.columns.find((c) => c.status === fromCol.status)!
    const nextToCol = next.columns.find((c) => c.status === toStatus)!
    const fromIndex = nextFromCol.tasks.findIndex((t) => t.id === activeId)
    const [moving] = nextFromCol.tasks.splice(fromIndex, 1)

    if (nextFromCol === nextToCol) {
      // Same column: arrayMove to handle index shift after splice.
      const adjustedIndex = toIndex > fromIndex ? toIndex - 1 : toIndex
      nextFromCol.tasks.splice(adjustedIndex, 0, moving)
    } else {
      nextToCol.tasks.splice(toIndex, 0, moving)
    }
    setBoard(next)

    try {
      const updated = await moveTask({
        id: activeId,
        status: toStatus,
        position: toIndex,
        filter,
      })
      setBoard(updated)
    } catch (err) {
      setBoard(previous)
      const msg = err instanceof Error ? err.message : "Failed to move task"
      toast.error(msg)
    }
  }

  return (
    <div className="w-full min-w-0 space-y-4 overflow-x-hidden p-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold tracking-tight">Tasks</h1>
        <Tabs value={filter} onValueChange={handleFilterChange}>
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="incomplete">Incomplete</TabsTrigger>
          </TabsList>
        </Tabs>
      </header>

      {loading ? (
        <BoardSkeleton />
      ) : error ? (
        <BoardError
          message={error}
          onRetry={() => {
            setLoading(true)
            fetchBoard(filter)
              .then((data) => setBoard(data))
              .catch((err: Error) => setError(err.message))
              .finally(() => setLoading(false))
          }}
        />
      ) : board ? (
        <BoardView
          board={board}
          collapsed={collapsed}
          onToggleCollapse={handleToggleCollapse}
          onDragEnd={handleDragEnd}
          sensors={sensors}
        />
      ) : null}
    </div>
  )
}

function BoardView({
  board,
  collapsed,
  onToggleCollapse,
  onDragEnd,
  sensors,
}: {
  board: BoardResponse
  collapsed: Set<string>
  onToggleCollapse: (status: string) => void
  onDragEnd: (event: DragEndEvent) => void
  sensors: ReturnType<typeof useSensors>
}) {
  const totalTasks = board.columns.reduce((acc, c) => acc + c.tasks.length, 0)
  if (totalTasks === 0) {
    return (
      <div className="rounded-lg border border-dashed bg-muted/30 p-12 text-center">
        <p className="text-sm text-muted-foreground">No tasks yet.</p>
      </div>
    )
  }
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragEnd={onDragEnd}
    >
      <div className="flex flex-col gap-4 md:flex-row md:items-stretch md:gap-3 md:overflow-x-auto md:pb-2">
        {board.columns.map((column) => (
          <KanbanColumn
            key={column.status}
            column={column}
            collapsed={collapsed.has(column.status)}
            onToggleCollapse={onToggleCollapse}
          />
        ))}
      </div>
    </DndContext>
  )
}

function BoardSkeleton() {
  return (
    <div className="flex flex-col gap-4 md:flex-row md:gap-3 md:overflow-x-auto">
      {[0, 1, 2, 3].map((i) => (
        <div
          key={i}
          className="flex w-full shrink-0 flex-col gap-2 rounded-lg border bg-muted/30 p-3 md:w-80"
        >
          <Skeleton className="h-5 w-24" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      ))}
    </div>
  )
}

function BoardError({
  message,
  onRetry,
}: {
  message: string
  onRetry: () => void
}) {
  return (
    <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-6">
      <p className="text-sm font-medium text-destructive">
        Failed to load board
      </p>
      <p className="mt-1 text-xs text-muted-foreground">{message}</p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="mt-3"
        onClick={onRetry}
      >
        Retry
      </Button>
    </div>
  )
}
