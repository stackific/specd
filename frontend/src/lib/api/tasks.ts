// API wrapper + types for the /tasks pages. Mirrors the JSON shapes emitted
// by cmd/api.go (apiBoardResponse, apiTaskDetailResponse) and the Go types
// in cmd/get_task.go and cmd/list_specs.go. Keep field names in sync with
// the Go `json:"..."` tags — this is the SPA's contract with the server.
import { fetchJSON } from "@/lib/api"

// ---------------------------------------------------------------------------
// Board (GET /api/tasks/board, POST /api/tasks/move)
// ---------------------------------------------------------------------------

export type BoardFilter = "all" | "incomplete"

export interface BoardCard {
  id: string
  spec_id: string
  title: string
  summary: string
  position: number
}

export interface BoardColumn {
  status: string
  label: string
  tasks: Array<BoardCard>
}

export interface BoardResponse {
  filter: BoardFilter
  stages: Array<string>
  columns: Array<BoardColumn>
}

export function fetchBoard(filter: BoardFilter): Promise<BoardResponse> {
  return fetchJSON<BoardResponse>(
    `/api/tasks/board?filter=${encodeURIComponent(filter)}`
  )
}

export interface MoveTaskRequest {
  id: string
  status: string
  position: number
  filter: BoardFilter
}

export function moveTask(req: MoveTaskRequest): Promise<BoardResponse> {
  return fetchJSON<BoardResponse>("/api/tasks/move", {
    method: "POST",
    body: JSON.stringify(req),
  })
}

// ---------------------------------------------------------------------------
// Task detail (GET /api/tasks/{id}, POST /api/tasks/{id}/criteria/{n}/toggle)
// ---------------------------------------------------------------------------

export interface TaskCriterion {
  position: number
  text: string
  checked: number
}

export interface TaskResponse {
  id: string
  spec_id: string
  title: string
  status: string
  summary: string
  body: string
  path: string
  position: number
  linked_tasks: Array<string>
  depends_on: Array<string>
  criteria: Array<TaskCriterion>
  created_by?: string
  updated_by?: string
  content_hash: string
  created_at: string
  updated_at: string
}

export interface ParentSpec {
  id: string
  title: string
  type: string
  summary: string
  position: number
  created_at: string
  updated_at: string
}

// TaskRef is the resolved-reference shape used for `depends_on_refs` and
// `linked_task_refs` on the detail payload. `summary` is the one-line
// snippet stored on the task (frontmatter `summary:`), included so the UI
// can render a title-plus-snippet row without a second fetch.
export interface TaskRef {
  id: string
  title: string
  summary: string
}

export interface TaskDetailResponse {
  task: TaskResponse
  parent_spec: ParentSpec | null
  status_label: string
  body_clean: string
  completed_count: number
  total_criteria: number
  depends_on_refs: Array<TaskRef>
  linked_task_refs: Array<TaskRef>
}

export function fetchTaskDetail(id: string): Promise<TaskDetailResponse> {
  return fetchJSON<TaskDetailResponse>(`/api/tasks/${encodeURIComponent(id)}`)
}

export function toggleCriterion(
  id: string,
  position: number
): Promise<TaskResponse> {
  return fetchJSON<TaskResponse>(
    `/api/tasks/${encodeURIComponent(id)}/criteria/${position}/toggle`,
    { method: "POST" }
  )
}

// setTaskDependsOn replaces the COMPLETE dependency set for a task. Pass the
// new full list (not a delta) — empty array clears all deps. Returns the
// freshly-loaded TaskDetailResponse so the page can re-render off the
// response without a second GET. Used by both the "remove dependency" X
// icon and the multi-select picker dialog.
export function setTaskDependsOn(
  id: string,
  dependsOn: Array<string>
): Promise<TaskDetailResponse> {
  return fetchJSON<TaskDetailResponse>(
    `/api/tasks/${encodeURIComponent(id)}/depends_on`,
    {
      method: "PUT",
      body: JSON.stringify({ depends_on: dependsOn }),
    }
  )
}

// ---------------------------------------------------------------------------
// Task list (GET /api/tasks)
// ---------------------------------------------------------------------------

// ListTaskItem mirrors the Go ListTaskItem struct used by /api/tasks.
// Used by the dependency picker dialog to populate the candidate list.
export interface ListTaskItem {
  id: string
  spec_id: string
  title: string
  status: string
  summary: string
  position: number
  created_at: string
  updated_at: string
}

export interface ListTasksResponse {
  items: Array<ListTaskItem>
  page: number
  page_size: number
  total_count: number
  total_pages: number
}

// listTasks fetches a single page of tasks. The picker dialog uses a large
// page size (500 by default) so all candidate tasks are available client-side
// for fast filtering; if a project ever exceeds that, this will need a
// proper paginated/searchable variant.
export function listTasks(
  opts: { pageSize?: number; signal?: AbortSignal } = {}
): Promise<ListTasksResponse> {
  const params = new URLSearchParams()
  params.set("page_size", String(opts.pageSize ?? 500))
  return fetchJSON<ListTasksResponse>(`/api/tasks?${params.toString()}`, {
    signal: opts.signal,
  })
}

// ---------------------------------------------------------------------------
// Delete (DELETE /api/tasks/{id})
// ---------------------------------------------------------------------------

// DeleteTaskResponse mirrors cmd/delete_task.go DeleteTaskResponse — keep
// field names in lockstep with the Go `json:"..."` tags.
export interface DeleteTaskResponse {
  id: string
  spec_id: string
  deleted: boolean
  path: string
}

// deleteTask removes a task on the server (DB + filesystem). Callers are
// responsible for any UI confirmation; this helper is a plain HTTP wrapper.
// The server returns the deleted record so the UI can surface a confirmation
// (e.g. "Deleted TASK-3 from SPEC-1").
export function deleteTask(id: string): Promise<DeleteTaskResponse> {
  return fetchJSON<DeleteTaskResponse>(`/api/tasks/${encodeURIComponent(id)}`, {
    method: "DELETE",
  })
}
