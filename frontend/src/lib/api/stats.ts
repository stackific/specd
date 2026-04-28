import { fetchJSON } from "@/lib/api"

export type StatsResponse = {
  tasks_total: number
  tasks_done: number
  specs: number
  kb_docs: number
}

export async function getStats(signal?: AbortSignal): Promise<StatsResponse> {
  return fetchJSON<StatsResponse>("/api/stats", { signal })
}
