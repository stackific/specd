import { fetchJSON } from "@/lib/api"

export type StartpageChoice = {
  title: string
  route: string
}

export type ProjectMeta = {
  name: string
  spec_types: Array<string>
  task_stages: Array<string>
  default_page_size: number
}

export type MetaResponse = {
  default_route: string
  project: ProjectMeta
  startpage_choices: Array<StartpageChoice>
}

export async function getMeta(signal?: AbortSignal): Promise<MetaResponse> {
  return fetchJSON<MetaResponse>("/api/meta", { signal })
}
