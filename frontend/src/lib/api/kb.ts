import { fetchJSON } from "@/lib/api"

export interface KBDocSummary {
  id: string
  title: string
  summary: string
  source_type: string
  path: string
  added_at: string
  added_by: string
}

export interface KBListResponse {
  items: Array<KBDocSummary>
}

// The Go detail handler returns Go-default JSON keys (no struct tags on
// KBDocDetail / KBChunkDetail), so the field names are PascalCase here.
export interface KBDocDetail {
  ID: string
  Title: string
  Summary: string
  SourceType: string
  Path: string
  AddedAt: string
  AddedBy: string
}

export interface KBChunkDetail {
  Position: number
  Summary: string
  Text: string
}

export interface KBDetailResponse {
  Doc: KBDocDetail
  Chunks: Array<KBChunkDetail> | null
}

export function fetchKBList(signal?: AbortSignal): Promise<KBListResponse> {
  return fetchJSON<KBListResponse>("/api/kb", { signal })
}

export function fetchKBDetail(
  id: string,
  signal?: AbortSignal
): Promise<KBDetailResponse> {
  return fetchJSON<KBDetailResponse>(`/api/kb/${encodeURIComponent(id)}`, {
    signal,
  })
}
