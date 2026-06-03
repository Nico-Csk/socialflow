import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { MeProvider, useMe } from '@/context/MeContext'

// Mock apiClient module BEFORE any imports that use it transitively
const mockGet = vi.fn()
const mockPost = vi.fn()
const mockSetUnauthorizedHandler = vi.fn()
const mockResetUnauthorizedHandler = vi.fn()
const mockResetUnauthorizedDebounce = vi.fn()

vi.mock('@/lib/apiClient', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
  setUnauthorizedHandler: (...args: unknown[]) =>
    mockSetUnauthorizedHandler(...args),
  resetUnauthorizedHandler: (...args: unknown[]) =>
    mockResetUnauthorizedHandler(...args),
  resetUnauthorizedDebounce: (...args: unknown[]) =>
    mockResetUnauthorizedDebounce(...args),
}))

// Consumer component that reads from MeContext to verify exposed state
function TestConsumer() {
  const { user, loading, error, login, logout, switchWorkspace, reauthenticate } = useMe()
  return (
    <div>
      <span data-testid="loading">{String(loading)}</span>
      <span data-testid="user-email">{user?.email ?? 'no-user'}</span>
      <span data-testid="user-workspace">{user?.active_workspace_id ?? 'no-workspace'}</span>
      <span data-testid="user-role">{user?.role ?? 'no-role'}</span>
      <span data-testid="error">{error ?? 'no-error'}</span>
      <span data-testid="reauth-result">-</span>
      <button
        data-testid="login-btn"
        onClick={() =>
          login({ id: '99', email: 'manual@test.com', active_workspace_id: 'ws1', role: 'admin' })
        }
      >
        Login Manually
      </button>
      <button
        data-testid="reauth-btn"
        onClick={async () => {
          const ok = await reauthenticate()
          const el = document.querySelector('[data-testid="reauth-result"]')
          if (el) el.textContent = String(ok)
        }}
      >
        Reauthenticate
      </button>
      <button
        data-testid="logout-btn"
        onClick={async () => {
          await logout()
        }}
      >
        Logout
      </button>
      <button
        data-testid="switch-btn"
        onClick={async () => {
          await switchWorkspace('ws2')
        }}
      >
        Switch Workspace
      </button>
    </div>
  )
}

function renderProvider() {
  return render(
    <MeProvider>
      <TestConsumer />
    </MeProvider>,
  )
}

describe('MeContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('registers unauthorized handler on mount and cleans up on unmount', async () => {
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    const { unmount } = renderProvider()

    await waitFor(() => {
      expect(mockSetUnauthorizedHandler).toHaveBeenCalledTimes(1)
      expect(mockSetUnauthorizedHandler).toHaveBeenCalledWith(
        expect.any(Function),
      )
    })

    // Not yet called
    expect(mockResetUnauthorizedHandler).not.toHaveBeenCalled()

    unmount()

    // Cleanup fires on unmount
    expect(mockResetUnauthorizedHandler).toHaveBeenCalledTimes(1)
  })

  it('unauthorized handler clears user state when invoked (auth-state transition)', async () => {
    // Seed with a logged-in user
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    renderProvider()

    // Wait for user to be loaded
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
    })

    // Capture the handler that MeProvider registered via setUnauthorizedHandler
    expect(mockSetUnauthorizedHandler).toHaveBeenCalledTimes(1)
    const handler = mockSetUnauthorizedHandler.mock.calls[0][0] as () => void

    // Simulate a 401 response from any API call — the interceptor fires
    act(() => {
      handler()
    })

    // User state must be cleared (setUser(null))
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Loading should remain false after the interceptor fires (no re-fetch)
    expect(screen.getByTestId('loading')).toHaveTextContent('false')
    // Error should be null — interceptor only clears user, doesn't set error
    expect(screen.getByTestId('error')).toHaveTextContent('no-error')
  })

  it('resetUnauthorizedHandler on unmount prevents stale handler from firing', async () => {
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    const { unmount } = renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
    })

    // Unmount the provider — this should call resetUnauthorizedHandler
    unmount()
    expect(mockResetUnauthorizedHandler).toHaveBeenCalledTimes(1)

    // After unmount, the handler is cleaned up; any future 401s will not
    // trigger the stale closure. Verified via resetUnauthorizedHandler call.
  })

  it('triggers refresh() on mount and exposes user state via useMe()', async () => {
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    renderProvider()

    // Verify refresh() was called (at least once; React StrictMode may call twice)
    expect(mockGet).toHaveBeenCalledWith('/me')

    // Wait for user state to be exposed
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
    })

    expect(screen.getByTestId('loading')).toHaveTextContent('false')
    expect(screen.getByTestId('error')).toHaveTextContent('no-error')
  })

  it('clears user state when refresh returns unauthorized', async () => {
    mockGet.mockResolvedValue({
      error: { code: 'unauthorized', message: 'Not authenticated' },
    })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    expect(screen.getByTestId('loading')).toHaveTextContent('false')
  })

  it('sets error state when refresh returns non-unauthorized error', async () => {
    mockGet.mockResolvedValue({
      error: { code: 'server_error', message: 'Internal error' },
    })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('error')).toHaveTextContent('Internal error')
    })
  })

  it('login() updates user state immediately without API call', async () => {
    mockGet.mockResolvedValue({ data: null }) // initial refresh returns no user

    renderProvider()

    // Wait for initial refresh to settle
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Click login button to manually set user
    screen.getByTestId('login-btn').click()

    // Wait for React state update to propagate to DOM
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('manual@test.com')
    })
    // login() clears error
    expect(screen.getByTestId('error')).toHaveTextContent('no-error')
  })

  // ── reauthenticate() ──────────────────────────────────────────

  it('reauthenticate() resets debounce, fetches /me, sets user on success, returns true', async () => {
    mockGet.mockResolvedValue({ data: null }) // initial refresh: no user

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Set up /me response for reauthenticate call
    mockGet.mockResolvedValue({
      data: { id: 'r1', email: 'reauth@test.com', active_workspace_id: 'ws2', role: 'editor' },
    })

    screen.getByTestId('reauth-btn').click()

    await waitFor(() => {
      // reauthenticate calls resetUnauthorizedDebounce BEFORE fetching /me
      expect(mockResetUnauthorizedDebounce).toHaveBeenCalled()
    })

    // Verify /me was called (refresh → apiClient.get('/me'))
    const meCalls = mockGet.mock.calls.filter((c: unknown[]) => c[0] === '/me')
    expect(meCalls.length).toBeGreaterThanOrEqual(1)

    // User state updated with reauth data
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('reauth@test.com')
    })

    // reauthenticate returns true
    expect(screen.getByTestId('reauth-result')).toHaveTextContent('true')
    expect(screen.getByTestId('error')).toHaveTextContent('no-error')
  })

  it('reauthenticate() returns false when /me returns server error, user=null, error set', async () => {
    mockGet.mockResolvedValue({ data: null })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Set /me to return a server error for reauthenticate
    mockGet.mockResolvedValue({
      error: { code: 'server_error', message: 'Internal server error' },
    })

    screen.getByTestId('reauth-btn').click()

    // debounce is still reset
    await waitFor(() => {
      expect(mockResetUnauthorizedDebounce).toHaveBeenCalled()
    })

    // User remains null
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Error message surfaced
    expect(screen.getByTestId('error')).toHaveTextContent('Internal server error')
    // reauthenticate returns false
    expect(screen.getByTestId('reauth-result')).toHaveTextContent('false')
  })

  // ── reauthenticate() unauthorized branch (R1 scenario gap) ─────

  it('reauthenticate() returns false when /me returns unauthorized, user=null, error stays null', async () => {
    // Seed with an initial user so we can observe clearing
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
    })

    // Set /me to return unauthorized for the reauthenticate call
    mockGet.mockResolvedValue({
      error: { code: 'unauthorized', message: 'Not authenticated' },
    })

    screen.getByTestId('reauth-btn').click()

    // debounce reset is called before /me
    await waitFor(() => {
      expect(mockResetUnauthorizedDebounce).toHaveBeenCalled()
    })

    // User cleared to null (unauthorized → session gone)
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })

    // Error stays null — unauthorized is not a user-facing error
    expect(screen.getByTestId('error')).toHaveTextContent('no-error')
    // reauthenticate returns false
    expect(screen.getByTestId('reauth-result')).toHaveTextContent('false')
  })

  // ── R5: Existing Behaviors Preserved ────────────────────────────

  it('logout() calls POST /auth/logout and clears user state', async () => {
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
    })

    // Set up logout API response
    mockPost.mockResolvedValue({ data: undefined }) // 204-like

    screen.getByTestId('logout-btn').click()

    // Verify POST /auth/logout was called
    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith('/auth/logout')
    })

    // User state is cleared
    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user')
    })
  })

  it('switchWorkspace() calls POST /workspaces/switch and updates local user state', async () => {
    mockGet.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', active_workspace_id: 'ws1', role: 'member' },
    })

    renderProvider()

    await waitFor(() => {
      expect(screen.getByTestId('user-email')).toHaveTextContent('user@test.com')
      expect(screen.getByTestId('user-workspace')).toHaveTextContent('ws1')
      expect(screen.getByTestId('user-role')).toHaveTextContent('member')
    })

    // Set up switch API response
    mockPost.mockResolvedValue({
      data: { active_workspace_id: 'ws2', role: 'editor' },
    })

    screen.getByTestId('switch-btn').click()

    // Verify POST /workspaces/switch was called with correct payload
    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith('/workspaces/switch', {
        workspace_id: 'ws2',
      })
    })

    // User state updated with new workspace and role
    await waitFor(() => {
      expect(screen.getByTestId('user-workspace')).toHaveTextContent('ws2')
      expect(screen.getByTestId('user-role')).toHaveTextContent('editor')
    })
  })
})
