import { describe, it, expect, vi, beforeEach } from 'vitest'
import apiClient, {
  ApiClient,
  setUnauthorizedHandler,
  resetUnauthorizedHandler,
  resetUnauthorizedDebounce,
} from '@/lib/apiClient'

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  describe('base URL construction', () => {
    it('constructs URL from /api base and path', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: { id: '1' } }),
        text: () => Promise.resolve(JSON.stringify({ data: { id: '1' } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/me')

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/me',
        expect.objectContaining({ method: 'GET' }),
      )
    })
  })

  describe('credentials inclusion', () => {
    it('includes credentials: include in every request', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: { ok: true } }),
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/status')

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/status',
        expect.objectContaining({ credentials: 'include' }),
      )
    })
  })

  describe('JSON body serialization', () => {
    it('serializes body to JSON for POST requests', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: { created: true } }),
        text: () =>
          Promise.resolve(JSON.stringify({ data: { created: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      const body = { email: 'test@example.com', password: 'secret' }
      await apiClient.post('/auth/login', body)

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/auth/login',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(body),
        }),
      )
    })

    it('does NOT serialize body for GET requests', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: [] }),
        text: () => Promise.resolve(JSON.stringify({ data: [] })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/items')

      const callArgs = fetchMock.mock.calls[0][1] as RequestInit
      expect(callArgs.body).toBeUndefined()
    })
  })

  describe('204 handling', () => {
    it('returns { data: undefined } for 204 No Content responses', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        json: vi.fn(), // should NOT be called
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.delete<null>('/items/1')

      expect(result).toEqual({ data: undefined })
    })
  })

  describe('error payload mapping', () => {
    it('maps error response to ApiResponse with error object', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: () =>
          Promise.resolve({
            error: { code: 'invalid_input', message: 'Email is required' },
          }),
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'invalid_input', message: 'Email is required' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.post('/auth/login', { email: '' })

      expect(result).toEqual({
        error: expect.objectContaining({
          code: 'invalid_input',
          message: 'Email is required',
        }),
      })
    })

    it('provides fallback error when response has no error body', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(JSON.stringify({})),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/failing-endpoint')

      expect(result.error).toEqual(
        expect.objectContaining({
          code: 'unknown',
          message: 'Request failed',
        }),
      )
    })
  })

  describe('successful data extraction', () => {
    it('extracts data field from JSON response', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () =>
          Promise.resolve({ data: { id: '42', email: 'user@test.com' } }),
        text: () =>
          Promise.resolve(
            JSON.stringify({ data: { id: '42', email: 'user@test.com' } }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get<{ id: string; email: string }>('/me')

      expect(result.data).toEqual({ id: '42', email: 'user@test.com' })
    })
  })

  // ── R1: Network Error Envelope ──────────────────────────────────
  describe('network error envelope', () => {
    it('returns error envelope instead of throwing on fetch rejection', async () => {
      const networkErr = new Error('Failed to fetch')
      const fetchMock = vi.fn().mockRejectedValue(networkErr)
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/test')

      expect(result.error).toEqual({
        code: 'network',
        message: 'Network error',
        details: String(networkErr),
      })
    })

    it('returns network error envelope for DNS failure', async () => {
      const fetchMock = vi
        .fn()
        .mockRejectedValue(new TypeError('Failed to fetch'))
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.post('/submit', { x: 1 })

      expect(result.error).toEqual({
        code: 'network',
        message: 'Network error',
        details: expect.stringContaining('Failed to fetch') as unknown,
      })
    })
  })

  // ── R2: Safe JSON Parsing ───────────────────────────────────────
  describe('safe JSON parsing', () => {
    it('returns parse error for HTML proxy response body', async () => {
      const htmlBody =
        '<html><body><h1>502 Bad Gateway</h1></body></html>'
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        text: () => Promise.resolve(htmlBody),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/test')

      expect(result.error).toMatchObject({
        code: 'parse',
        message: 'Invalid JSON',
      })
      expect(result.error?.details).toMatchObject({
        status: 502,
        preview: expect.stringContaining('502 Bad Gateway') as unknown,
      })
    })

    it('returns { data: undefined } for empty 200 response', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(''),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/test')

      expect(result).toEqual({ data: undefined })
    })

    it('handles valid JSON non-ok response correctly', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: {
                code: 'invalid_input',
                message: 'Email is required',
              },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.post('/test', { email: '' })

      expect(result.error).toMatchObject({
        code: 'invalid_input',
        message: 'Email is required',
      })
    })
  })

  // ── R3: Header Merge ────────────────────────────────────────────
  describe('header merge', () => {
    it('merges caller headers preserving Content-Type default', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/test', {
        headers: { 'X-Custom': 'value' },
      })

      const config = fetchMock.mock.calls[0][1] as RequestInit
      expect(config.headers).toBeInstanceOf(Headers)
      const h = config.headers as Headers
      expect(h.get('Content-Type')).toBe('application/json')
      expect(h.get('X-Custom')).toBe('value')
    })

    it('allows caller to override Content-Type', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.post(
        '/upload',
        {},
        {
          headers: { 'Content-Type': 'multipart/form-data' },
        },
      )

      const config = fetchMock.mock.calls[0][1] as RequestInit
      const h = config.headers as Headers
      expect(h.get('Content-Type')).toBe('multipart/form-data')
    })
  })

  // ── R3 (Credentials Lock) ───────────────────────────────────────
  describe('credentials lock', () => {
    it('ignores caller credentials override attempt', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/test', { credentials: 'omit' } as RequestInit)

      expect(fetchMock).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ credentials: 'include' }),
      )
    })

    it('includes credentials: include when no options passed', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/test')

      expect(fetchMock).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ credentials: 'include' }),
      )
    })
  })

  // ── R4: Error Details Preservation ──────────────────────────────
  describe('error details preservation', () => {
    it('preserves backend error.details in ApiError', async () => {
      const detailsObj = { fields: { email: 'Required', age: 'Must be > 0' } }
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: {
                code: 'invalid_input',
                message: 'Validation failed',
                details: detailsObj,
              },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.post('/test', {})

      expect(result.error).toMatchObject({
        code: 'invalid_input',
        message: 'Validation failed',
        details: detailsObj,
      })
    })

    it('returns undefined details when backend omits them', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'internal' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/test')

      expect(result.error).toMatchObject({
        code: 'internal',
        message: 'Request failed',
        details: undefined,
      })
    })
  })

  // ── R5: Configurable Base URL ───────────────────────────────────
  describe('configurable base URL', () => {
    it('reads VITE_API_BASE_URL env var when set', async () => {
      vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:3000/api')

      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      // Create fresh instance to pick up the env var
      const client = new ApiClient()
      await client.get('/me')

      expect(fetchMock).toHaveBeenCalledWith(
        'http://localhost:3000/api/me',
        expect.any(Object),
      )

      vi.unstubAllEnvs()
    })

    it('falls back to /api when env var is unset', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      const client = new ApiClient()
      await client.get('/me')

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/me',
        expect.any(Object),
      )
    })

    it('strips trailing slash from base URL', async () => {
      vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:3000/api/')

      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ data: { ok: true } })),
      })
      vi.stubGlobal('fetch', fetchMock)

      const client = new ApiClient()
      await client.get('/me')

      expect(fetchMock).toHaveBeenCalledWith(
        'http://localhost:3000/api/me',
        expect.any(Object),
      )

      vi.unstubAllEnvs()
    })
  })

  // ── R6: 401 Interceptor ─────────────────────────────────────────
  describe('401 interceptor', () => {
    beforeEach(() => {
      resetUnauthorizedHandler()
    })

    it('fires registered handler on 401 response and returns concrete error shape', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      const result = await apiClient.get('/me')

      expect(handler).toHaveBeenCalledTimes(1)
      // Error returned to caller with full concrete shape
      expect(result.error).toEqual({
        code: 'unauthorized',
        message: 'Session expired',
        status: 401,
      })
    })

    it('debounces parallel 401s — handler fires exactly once', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      await Promise.all([
        apiClient.get('/me'),
        apiClient.get('/workspaces'),
        apiClient.get('/projects'),
      ])

      expect(handler).toHaveBeenCalledTimes(1)
    })

    it('does NOT fire handler on non-401 errors', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'internal', message: 'Server error' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/test')

      expect(handler).not.toHaveBeenCalled()
    })

    it('resetUnauthorizedHandler clears the handler', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)
      resetUnauthorizedHandler()

      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', fetchMock)

      await apiClient.get('/me')

      expect(handler).not.toHaveBeenCalled()
    })

    it('resetUnauthorizedHandler resets the debounce flag', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      // First 401 — fires handler
      const mock401 = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', mock401)
      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(1)

      // Reset — allows re-firing
      resetUnauthorizedHandler()
      setUnauthorizedHandler(handler)

      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(2)
    })

    it('resetUnauthorizedDebounce resets handlerFired only, preserving the registered handler', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      const mock401 = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', mock401)

      // First 401 — handler fires once, debounce latches
      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(1)

      // Reset debounce ONLY (not the handler itself)
      resetUnauthorizedDebounce()

      // Second 401 — handler fires AGAIN without re-registration
      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(2)
      // Handler reference was never replaced — it's the same function
    })

    it('resetUnauthorizedDebounce does NOT clear the unauthorized handler', async () => {
      const handler = vi.fn()
      setUnauthorizedHandler(handler)

      const mock401 = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          Promise.resolve(
            JSON.stringify({
              error: { code: 'unauthorized', message: 'Session expired' },
            }),
          ),
      })
      vi.stubGlobal('fetch', mock401)

      // Fire first 401
      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(1)

      // Reset debounce
      resetUnauthorizedDebounce()

      // Fire second 401 — handler still registered, fires again
      await apiClient.get('/me')
      expect(handler).toHaveBeenCalledTimes(2)
    })
  })
})
