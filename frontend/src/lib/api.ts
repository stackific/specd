export class ApiError extends Error {
  status: number
  code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export async function fetchJSON<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
  })
  if (!res.ok) {
    let message = res.statusText
    let code: string | undefined
    try {
      const body = await res.json()
      if (typeof body?.error === "string") message = body.error
      if (typeof body?.code === "string") code = body.code
    } catch {
      /* noop */
    }
    throw new ApiError(res.status, message, code)
  }
  if (res.status === 204) return undefined as unknown as T
  return (await res.json()) as T
}
