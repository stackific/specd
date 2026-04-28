import { fetchJSON } from "@/lib/api"

export type SpecsView = "grouped" | "cards" | "flat"

export const SPECS_TYPE_ALL = "all"

export type ListSpecItem = {
  id: string
  title: string
  type: string
  summary: string
  position: number
  created_at: string
  updated_at: string
}

export type SpecsGroup = {
  Type: string
  Items: Array<ListSpecItem>
  Total: number
}

export type ListSpecsResponse = {
  view: SpecsView
  type: string
  types: Array<string>
  items: Array<ListSpecItem>
  groups: Array<SpecsGroup>
  page: number
  page_size: number
  total_count: number
  total_pages: number
}

export type SpecClaim = {
  position: number
  text: string
}

export type SpecTaskCriterion = {
  position: number
  text: string
  checked: number
}

export type SpecTask = {
  id: string
  title: string
  status: string
  summary: string
  criteria: Array<SpecTaskCriterion>
}

// SpecRef is the resolved-reference shape used for `linked_specs_refs`.
// `summary` is the one-line snippet stored on the spec (frontmatter
// `summary:`), included so the UI can render a title-plus-snippet row
// without a second fetch.
export interface SpecRef {
  id: string
  title: string
  summary: string
}

export type GetSpecResponse = {
  id: string
  title: string
  type: string
  summary: string
  body: string
  path: string
  position: number
  linked_specs: Array<string>
  linked_specs_refs: Array<SpecRef>
  claims: Array<SpecClaim>
  tasks: Array<SpecTask>
  created_by?: string
  updated_by?: string
  content_hash: string
  created_at: string
  updated_at: string
}

export type ListSpecsParams = {
  view?: SpecsView
  type?: string
  page?: number
  pageSize?: number
  signal?: AbortSignal
}

export async function listSpecs({
  view = "grouped",
  type = SPECS_TYPE_ALL,
  page = 1,
  pageSize = 20,
  signal,
}: ListSpecsParams = {}): Promise<ListSpecsResponse> {
  const params = new URLSearchParams({
    view,
    type,
    page: String(page),
    page_size: String(pageSize),
  })
  return fetchJSON<ListSpecsResponse>(`/api/specs?${params.toString()}`, {
    signal,
  })
}

export async function getSpec(
  id: string,
  signal?: AbortSignal
): Promise<GetSpecResponse> {
  return fetchJSON<GetSpecResponse>(`/api/specs/${encodeURIComponent(id)}`, {
    signal,
  })
}

// unlinkSpec removes the bidirectional link between two specs and returns
// the freshly-loaded source-spec detail so the page can re-render off the
// response (no second GET). Both spec markdown files are rewritten on the
// server so frontmatter matches the DB.
export async function unlinkSpec(
  id: string,
  linkedId: string
): Promise<GetSpecResponse> {
  return fetchJSON<GetSpecResponse>(
    `/api/specs/${encodeURIComponent(id)}/linked_specs/${encodeURIComponent(linkedId)}`,
    { method: "DELETE" }
  )
}

// setLinkedSpecs replaces the COMPLETE linked_specs set for a spec. Pass
// the new full list (not a delta); empty array clears all links. Returns
// the freshly-loaded spec detail. Mirrors `setTaskDependsOn` so picker UIs
// can reuse the same shape: one wrapper handles both add and remove paths.
export async function setLinkedSpecs(
  id: string,
  linkedSpecs: Array<string>
): Promise<GetSpecResponse> {
  return fetchJSON<GetSpecResponse>(
    `/api/specs/${encodeURIComponent(id)}/linked_specs`,
    {
      method: "PUT",
      body: JSON.stringify({ linked_specs: linkedSpecs }),
    }
  )
}
