import { fetchJSON } from "@/lib/api"

export type SearchKind = "all" | "spec" | "task" | "kb"

export type SearchResultKind = "spec" | "task" | "kb"

export type SearchResult = {
  kind: SearchResultKind
  id: string
  title: string
  summary?: string
  score: number
  match_type: string
}

export type SearchResponse = {
  query: string
  kind: string
  page: number
  page_size: number
  total_count: number
  total_pages: number
  items: Array<SearchResult>
  page_specs: Array<SearchResult>
  page_tasks: Array<SearchResult>
  page_kb: Array<SearchResult>
}

export type SearchParams = {
  q: string
  kind?: SearchKind
  page?: number
  pageSize?: number
  signal?: AbortSignal
}

export async function searchAll({
  q,
  kind = "all",
  page = 1,
  pageSize = 10,
  signal,
}: SearchParams): Promise<SearchResponse> {
  const params = new URLSearchParams({
    q,
    kind,
    page: String(page),
    page_size: String(pageSize),
  })
  return fetchJSON<SearchResponse>(`/api/search?${params.toString()}`, {
    signal,
  })
}

export function detailPathFor(kind: SearchResultKind, id: string): string {
  switch (kind) {
    case "spec":
      return `/specs/${id}`
    case "task":
      return `/tasks/${id}`
    case "kb":
      return `/kb/${id}`
  }
}

export function kindLabel(kind: SearchResultKind): string {
  switch (kind) {
    case "spec":
      return "Spec"
    case "task":
      return "Task"
    case "kb":
      return "KB"
  }
}
