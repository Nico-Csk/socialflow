// apiClient — centralized fetch wrapper for SocialFlow API.
// All requests include credentials (http-only cookie) by default.

export interface ApiError {
  code: string
  message: string
  details?: unknown
  status?: number
}

export interface ApiResponse<T> {
  data?: T
  error?: ApiError
}

// ── Base URL resolution (R5) ──────────────────────────────────────

export function resolveBaseUrl(): string {
  const base = import.meta.env.VITE_API_BASE_URL ?? '/api'
  return base.replace(/\/+$/, '')
}

// ── Header merging (R3) ───────────────────────────────────────────

export function mergeHeaders(callerHeaders?: HeadersInit): Headers {
  const headers = new Headers()
  headers.set('Content-Type', 'application/json')

  if (callerHeaders) {
    const caller = new Headers(callerHeaders)
    caller.forEach((value, key) => {
      headers.set(key, value)
    })
  }

  return headers
}

// ── Safe body parsing (R2) ────────────────────────────────────────

async function parseResponseBody(
  res: Response,
): Promise<{ rawText: string; parsed: unknown | null }> {
  const rawText = await res.text()
  try {
    const parsed = JSON.parse(rawText)
    return { rawText, parsed }
  } catch {
    return { rawText, parsed: null }
  }
}

// ── Error normalization (R2, R4) ──────────────────────────────────

function normalizeError(
  res: Response,
  parsed: unknown,
  rawText: string,
): ApiError {
  // Parse failure → parse error
  if (parsed === null) {
    const preview =
      rawText.length > 500 ? rawText.slice(0, 500) + '...' : rawText
    return {
      code: 'parse',
      message: 'Invalid JSON',
      details: { status: res.status, preview },
      status: res.status,
    }
  }

  // Backend error payload
  if (
    parsed !== null &&
    typeof parsed === 'object' &&
    'error' in parsed &&
    (parsed as { error: unknown }).error != null
  ) {
    const err = (parsed as { error: Record<string, unknown> }).error
    return {
      code: String(err.code ?? 'unknown'),
      message: String(err.message ?? 'Request failed'),
      details: 'details' in err ? err.details : undefined,
      status: res.status,
    }
  }

  // Fallback error
  return {
    code: 'unknown',
    message: 'Request failed',
    status: res.status,
  }
}

// ── 401 interceptor (R6) ─────────────────────────────────────────

export type UnauthorizedHandler = () => void

let unauthorizedHandler: UnauthorizedHandler | null = null
let handlerFired = false

export function setUnauthorizedHandler(handler: UnauthorizedHandler): void {
  unauthorizedHandler = handler
  handlerFired = false
}

export function resetUnauthorizedHandler(): void {
  unauthorizedHandler = null
  handlerFired = false
}

export function resetUnauthorizedDebounce(): void {
  handlerFired = false
}

// ── ApiClient ──────────────────────────────────────────────────────

export class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = resolveBaseUrl()) {
    this.baseUrl = baseUrl
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestInit,
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseUrl}${path}`

    const config: RequestInit = {
      method,
      credentials: 'include',
      headers: mergeHeaders(options?.headers),
    }

    if (body && method !== 'GET' && method !== 'HEAD') {
      config.body = JSON.stringify(body)
    }

    // Cherry-pick caller options that don't conflict with our defaults
    if (options) {
      const rest = { ...options }
      delete rest.headers
      delete rest.credentials
      Object.assign(config, rest)
    }

    let res: Response
    try {
      res = await fetch(url, config)
    } catch (err) {
      return {
        error: {
          code: 'network',
          message: 'Network error',
          details: String(err),
        },
      }
    }

    if (res.status === 204) {
      return { data: undefined }
    }

    const { rawText, parsed } = await parseResponseBody(res)

    if (!res.ok) {
      // Fire unauthorized handler on 401 (debounced)
      if (res.status === 401 && unauthorizedHandler && !handlerFired) {
        handlerFired = true
        unauthorizedHandler()
      }

      return { error: normalizeError(res, parsed, rawText) }
    }

    // Successful response: empty body → data: undefined
    if (parsed === null || parsed === undefined) {
      return { data: undefined as T }
    }

    const data = parsed as { data?: T }
    return { data: data.data !== undefined ? data.data : (parsed as T) }
  }

  get<T>(path: string, options?: RequestInit) {
    return this.request<T>('GET', path, undefined, options)
  }

  post<T>(path: string, body?: unknown, options?: RequestInit) {
    return this.request<T>('POST', path, body, options)
  }

  put<T>(path: string, body?: unknown, options?: RequestInit) {
    return this.request<T>('PUT', path, body, options)
  }

  patch<T>(path: string, body?: unknown, options?: RequestInit) {
    return this.request<T>('PATCH', path, body, options)
  }

  delete<T>(path: string, options?: RequestInit) {
    return this.request<T>('DELETE', path, undefined, options)
  }
}

export const apiClient = new ApiClient()
export default apiClient
