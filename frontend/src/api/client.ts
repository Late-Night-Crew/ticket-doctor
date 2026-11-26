function base(): string {
  const origin = import.meta.env.VITE_API_URL || window.location.origin
  return `${origin.replace(/\/$/, '')}/api`
}

export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    let message = res.statusText
    try {
      // The Go API sends RFC 7807 problem details ({ title, status, detail });
      // "error" is accepted too for older responses.
      const body = (await res.json()) as { detail?: string; error?: string }
      message = body.detail || body.error || message
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, message)
  }
  return (await res.json()) as T
}

export const post = <T>(path: string, body: unknown) =>
  api<T>(path, { method: 'POST', body: JSON.stringify(body) })
