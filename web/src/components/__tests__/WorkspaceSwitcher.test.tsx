import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// ── Mock spies (defined before vi.mock hoisting via vi.hoisted) ────
const mockSwitchWorkspace = vi.fn()
const mockRefresh = vi.fn()
const mockApiGet = vi.fn()

// ── Module mocks ──────────────────────────────────────────────────
vi.mock('@/lib/apiClient', () => ({
  default: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/context/MeContext', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/context/MeContext')>()
  return {
    ...actual,
    useMe: () => ({
      user: {
        id: 'u1',
        email: 'test@socialflow.dev',
        active_workspace_id: 'ws1',
        role: 'member',
      },
      loading: false,
      error: null,
      switchWorkspace: mockSwitchWorkspace,
      refresh: mockRefresh,
      reauthenticate: vi.fn(),
      login: vi.fn(),
      logout: vi.fn(),
    }),
  }
})

import WorkspaceSwitcher from '@/components/WorkspaceSwitcher'

// ── Fixtures ──────────────────────────────────────────────────────
const MOCK_WORKSPACES = [
  { id: 'ws1', name: 'Acme Corp' },
  { id: 'ws2', name: 'Startup Inc' },
  { id: 'ws3', name: 'Freelance' },
]

function setup() {
  mockApiGet.mockResolvedValue({ data: MOCK_WORKSPACES })
}

describe('WorkspaceSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── 1.1 Test scaffolding: renders workspace list ─────────────────

  it('renders workspace list loaded from API', async () => {
    setup()
    render(<WorkspaceSwitcher />)

    // Open dropdown so workspace items are visible
    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    expect(screen.getByText('Startup Inc')).toBeInTheDocument()
    expect(screen.getByText('Freelance')).toBeInTheDocument()
    expect(mockApiGet).toHaveBeenCalledWith('/workspaces')
  })

  it('shows active workspace in button label', async () => {
    setup()
    render(<WorkspaceSwitcher />)

    await waitFor(() => {
      // The button displays current workspace name (ws1 = Acme Corp)
      const trigger = screen.getByRole('button', { name: /Acme Corp/i })
      expect(trigger).toBeInTheDocument()
    })
  })

  // ── 1.2 Switch calls switchWorkspace, refresh is NOT called ─────

  it('calls switchWorkspace with target id and never calls refresh on success', async () => {
    setup()
    mockSwitchWorkspace.mockResolvedValue({
      active_workspace_id: 'ws2',
      role: 'editor',
    })

    render(<WorkspaceSwitcher />)

    // Open dropdown
    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // Click target workspace
    const targetBtn = await screen.findByRole('button', { name: /Startup Inc/i })
    await userEvent.click(targetBtn)

    // Assertions
    expect(mockSwitchWorkspace).toHaveBeenCalledTimes(1)
    expect(mockSwitchWorkspace).toHaveBeenCalledWith('ws2')
    // KEY ASSERTION: refresh MUST NOT be called after a successful switch
    expect(mockRefresh).not.toHaveBeenCalled()
    // Direct network assertion: no GET /me request was issued through any path
    const meCalls = mockApiGet.mock.calls.filter((call) => call[0] === '/me')
    expect(meCalls).toHaveLength(0)
  })

  // ── 1.3 Menu closes after successful switch ─────────────────────

  it('closes dropdown after successful workspace switch', async () => {
    setup()
    mockSwitchWorkspace.mockResolvedValue({
      active_workspace_id: 'ws2',
      role: 'editor',
    })

    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // Dropdown is open — workspaces are visible
    expect(screen.getByText('Startup Inc')).toBeInTheDocument()

    // Click a workspace to switch
    await userEvent.click(screen.getByRole('button', { name: /Startup Inc/i }))

    // Dropdown should close — workspace items removed from DOM
    await waitFor(() => {
      expect(screen.queryByText('Startup Inc')).not.toBeInTheDocument()
    })
  })

  // ── 1.4 Same-workspace click returns early ──────────────────────

  it('does nothing when clicking the already-active workspace', async () => {
    setup()
    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // Active workspace has "(active)" badge
    const activeBtn = screen.getByRole('button', { name: /Acme Corp.*active/i })
    await userEvent.click(activeBtn)

    // switchWorkspace must NOT be called
    expect(mockSwitchWorkspace).not.toHaveBeenCalled()
    // refresh must NOT be called
    expect(mockRefresh).not.toHaveBeenCalled()
    // Direct network: no GET /me request was issued
    const meCalls = mockApiGet.mock.calls.filter((call) => call[0] === '/me')
    expect(meCalls).toHaveLength(0)
    // dropdown must close
    await waitFor(() => {
      expect(screen.queryByText(/Startup Inc/)).not.toBeInTheDocument()
    })
  })

  // ── 1.5 Error path: switchWorkspace returns null ────────────────

  it('handles switch error gracefully without calling refresh', async () => {
    setup()
    mockSwitchWorkspace.mockResolvedValue(null) // error path

    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    await userEvent.click(screen.getByRole('button', { name: /Startup Inc/i }))

    // switchWorkspace was still called
    expect(mockSwitchWorkspace).toHaveBeenCalledWith('ws2')
    // refresh must NOT be called on error path either
    expect(mockRefresh).not.toHaveBeenCalled()
    // Direct network: no GET /me request was issued
    const meCalls = mockApiGet.mock.calls.filter((call) => call[0] === '/me')
    expect(meCalls).toHaveLength(0)
    // dropdown closes
    await waitFor(() => {
      expect(screen.queryByText(/Startup Inc/)).not.toBeInTheDocument()
    })
    // trigger button should be re-enabled (not disabled)
    const triggerAfter = screen.getByRole('button', { name: /Acme Corp/i })
    expect(triggerAfter).not.toBeDisabled()
  })

  // ── 1.6 Active workspace is highlighted in dropdown ─────────────

  it('highlights the active workspace in the dropdown menu', async () => {
    setup()
    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // The active workspace button should contain "(active)" text
    const activeBtn = screen.getByRole('button', { name: /Acme Corp.*active/i })
    expect(activeBtn).toBeInTheDocument()

    // Non-active workspaces do NOT have "(active)" badge
    const inactiveBtn = screen.getByRole('button', { name: /^Startup Inc$/ })
    expect(inactiveBtn).toBeInTheDocument()
    expect(inactiveBtn).not.toHaveTextContent('(active)')
  })

  // ── Edge: workspaces empty state ────────────────────────────────

  it('shows empty state when no workspaces are returned', async () => {
    mockApiGet.mockResolvedValue({ data: [] })
    render(<WorkspaceSwitcher />)

    // Open dropdown to see empty state message
    const trigger = await screen.findByRole('button', { name: /No workspace/i })
    await userEvent.click(trigger)

    expect(screen.getByText('No workspaces yet.')).toBeInTheDocument()
  })

  // ── Edge: API error loading workspaces ──────────────────────────

  it('handles API error loading workspaces gracefully', async () => {
    mockApiGet.mockResolvedValue({ error: { code: 'network', message: 'Offline' } })
    render(<WorkspaceSwitcher />)

    // Open dropdown to see empty state message
    const trigger = await screen.findByRole('button', { name: /No workspace/i })
    await userEvent.click(trigger)

    expect(screen.getByText('No workspaces yet.')).toBeInTheDocument()
  })

  // ── Regression: menu toggle behavior ────────────────────────────

  it('opens and closes dropdown when clicking the trigger button', async () => {
    setup()
    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })

    // Open
    await userEvent.click(trigger)
    expect(screen.getByText('Startup Inc')).toBeInTheDocument()

    // Close
    await userEvent.click(trigger)
    await waitFor(() => {
      expect(screen.queryByText('Startup Inc')).not.toBeInTheDocument()
    })
  })

  // ── Escape key closes dropdown ────────────────────────────────────

  it('closes dropdown when Escape key is pressed', async () => {
    setup()
    mockSwitchWorkspace.mockResolvedValue({
      active_workspace_id: 'ws2',
      role: 'editor',
    })

    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // Dropdown is open
    expect(screen.getByText('Startup Inc')).toBeInTheDocument()

    // Press Escape key
    await userEvent.keyboard('{Escape}')

    // Dropdown should close
    await waitFor(() => {
      expect(screen.queryByText('Startup Inc')).not.toBeInTheDocument()
    })

    // No side effects: refresh not called
    expect(mockRefresh).not.toHaveBeenCalled()
  })

  // ── Regression: button is disabled while switching ──────────────

  it('disables the trigger button while a switch is in progress', async () => {
    setup()
    // Create a promise we control to keep the switch pending
    let resolveSwitch: (value: { active_workspace_id: string; role: string }) => void
    const switchPromise = new Promise<{ active_workspace_id: string; role: string }>(
      (resolve) => {
        resolveSwitch = resolve
      },
    )
    mockSwitchWorkspace.mockReturnValue(switchPromise)

    render(<WorkspaceSwitcher />)

    const trigger = await screen.findByRole('button', { name: /Acme Corp/i })
    await userEvent.click(trigger)

    // Click target workspace — switch starts but doesn't resolve yet
    await userEvent.click(screen.getByRole('button', { name: /Startup Inc/i }))

    // Trigger button should be disabled during switch
    const triggerBtn = screen.getAllByRole('button', { name: /Acme Corp/i })[0]
    expect(triggerBtn).toBeDisabled()

    // Resolve the switch
    resolveSwitch!({ active_workspace_id: 'ws2', role: 'editor' })

    // Button should be re-enabled after switch completes (dropdown closes, so only one match)
    await waitFor(() => {
      const btn = screen.getByRole('button', { name: /Acme Corp/i })
      expect(btn).not.toBeDisabled()
    })

    // Refresh must still not be called
    expect(mockRefresh).not.toHaveBeenCalled()
  })
})
