import { fetchJSON } from "@/lib/api"

export type SetDefaultRouteResponse = {
  ok: boolean
  default_route: string
}

export async function setDefaultRoute(
  defaultRoute: string,
  signal?: AbortSignal
): Promise<SetDefaultRouteResponse> {
  return fetchJSON<SetDefaultRouteResponse>("/api/settings/default-route", {
    method: "POST",
    body: JSON.stringify({ default_route: defaultRoute }),
    signal,
  })
}
