/**
 * Thin fetch client for the CarePath API. Response types come from the
 * contract-generated schema.d.ts — never hand-write a shape that exists there.
 */

const apiBase = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function apiGet<TReturn>(path: string): Promise<TReturn> {
  return request(path, { method: 'GET' })
}

export async function apiPost<TReturn>(path: string, body: unknown): Promise<TReturn> {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

async function request<TReturn>(path: string, init: RequestInit): Promise<TReturn> {
  let response: Response
  try {
    response = await fetch(`${apiBase}${path}`, init)
  } catch (cause) {
    throw new ApiError(0, `เชื่อมต่อ API ไม่ได้ (${String(cause)})`)
  }
  if (!response.ok) {
    let detail = `${response.status}`
    try {
      const body = (await response.json()) as { error?: string }
      if (body.error) detail = body.error
    } catch {
      // Non-JSON error body — keep the status code as the detail.
    }
    throw new ApiError(response.status, detail)
  }
  return (await response.json()) as TReturn
}
