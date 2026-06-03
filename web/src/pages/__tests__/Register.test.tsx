import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '@/test/render'

// Mock navigation
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Mock apiClient — both register and login calls need to be observable
const mockPost = vi.fn()

vi.mock('@/lib/apiClient', () => ({
  default: {
    post: (...args: unknown[]) => mockPost(...args),
  },
}))

// Mock useMe for reauthenticate integration
const mockReauthenticate = vi.fn()

vi.mock('@/context/MeContext', () => ({
  useMe: () => ({
    reauthenticate: mockReauthenticate,
  }),
}))

import Register from '@/pages/Register'

describe('Register page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()
    mockReauthenticate.mockClear()
  })

  it('renders name, email, password fields, submit button, and login link', () => {
    renderWithRouter(<Register />)

    expect(screen.getByLabelText('Name')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in' })).toHaveAttribute('href', '/login')
  })

  it('displays error message when registration API returns an error', async () => {
    mockPost.mockResolvedValue({
      error: { code: 'email_taken', message: 'Email already registered' },
    })

    renderWithRouter(<Register />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'taken@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    await waitFor(() => {
      expect(screen.getByText('Email already registered')).toBeInTheDocument()
    })

    // reauthenticate is NOT called on registration error
    expect(mockReauthenticate).not.toHaveBeenCalled()
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('does NOT call POST /auth/login after successful registration', async () => {
    mockPost.mockResolvedValue({
      data: { id: '2', email: 'new@test.com', name: 'New User' },
    })
    mockReauthenticate.mockResolvedValue(true)

    renderWithRouter(<Register />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'new@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    // reauthenticate is called instead of manual login
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

    // /auth/login is NEVER called
    const loginCalls = mockPost.mock.calls.filter(
      (c: unknown[]) => c[0] === '/auth/login',
    )
    expect(loginCalls).toHaveLength(0)

    // Navigates to dashboard on success
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
    })
  })

  it('shows user-facing error when reauthenticate fails after registration', async () => {
    mockPost.mockResolvedValue({
      data: { id: '2', email: 'new@test.com', name: 'New User' },
    })
    mockReauthenticate.mockResolvedValue(false)

    renderWithRouter(<Register />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'new@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    // reauthenticate was called
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

    // Error message is displayed
    await waitFor(() => {
      expect(
        screen.getByText(
          'Account created but session not verified. Please log in.',
        ),
      ).toBeInTheDocument()
    })

    // Navigation does NOT happen
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('shows loading state while registration is in flight', async () => {
    mockPost.mockReturnValue(new Promise(() => {}))

    renderWithRouter(<Register />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'new@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Creating account...' })).toBeInTheDocument()
    })

    expect(screen.getByRole('button')).toBeDisabled()
  })

  // ── Ordering: navigation waits for reauthenticate resolution ────

  it('delays navigation until reauthenticate resolves after registration, proving ordering', async () => {
    mockPost.mockResolvedValue({
      data: { id: '2', email: 'new@test.com', name: 'New User' },
    })

    // Create a deferred promise for reauthenticate
    let resolveReauth!: (value: boolean) => void
    const reauthPromise = new Promise<boolean>((resolve) => {
      resolveReauth = resolve
    })
    mockReauthenticate.mockReturnValue(reauthPromise)

    renderWithRouter(<Register />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'new@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    // reauthenticate is called after registration succeeds
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

    // /auth/login is NEVER called
    const loginCalls = mockPost.mock.calls.filter(
      (c: unknown[]) => c[0] === '/auth/login',
    )
    expect(loginCalls).toHaveLength(0)

    // Navigation must NOT have happened yet — reauth still pending
    expect(mockNavigate).not.toHaveBeenCalled()

    // Now resolve reauthenticate
    resolveReauth(true)

    // Navigation fires only AFTER reauthenticate resolves
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
    })
  })
})
