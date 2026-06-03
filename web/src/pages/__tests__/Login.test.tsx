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

// Mock apiClient module BEFORE importing Login
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

import Login from '@/pages/Login'

describe('Login page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()
    mockReauthenticate.mockClear()
  })

  it('renders email field, password field, submit button, and register link', () => {
    renderWithRouter(<Login />)

    // Form fields
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument()

    // Register link
    expect(screen.getByRole('link', { name: 'Register' })).toHaveAttribute(
      'href',
      '/register',
    )
  })

  it('displays error message when API returns an error', async () => {
    mockPost.mockResolvedValue({
      error: { code: 'invalid_credentials', message: 'Invalid email or password' },
    })

    renderWithRouter(<Login />)

    // Fill and submit the form
    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'bad@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'wrong' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // Wait for error message to appear
    await waitFor(() => {
      expect(screen.getByText('Invalid email or password')).toBeInTheDocument()
    })

    // Verify API was called with correct payload
    expect(mockPost).toHaveBeenCalledWith('/auth/login', {
      email: 'bad@test.com',
      password: 'wrong',
    })
  })

  it('shows loading state on submit button while request is in flight', async () => {
    // Never-resolving promise to keep loading state active
    mockPost.mockReturnValue(new Promise(() => {}))

    renderWithRouter(<Login />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'user@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // Button text changes to loading state
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Logging in...' })).toBeInTheDocument()
    })

    // Button is disabled while loading
    expect(screen.getByRole('button')).toBeDisabled()
  })

  // ── reauthenticate integration ─────────────────────────────────

  it('calls reauthenticate() after successful login and navigates to /dashboard', async () => {
    mockPost.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', name: 'Test' },
    })
    mockReauthenticate.mockResolvedValue(true)

    renderWithRouter(<Login />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'user@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // reauthenticate is called after login succeeds
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

    // Navigation happens after reauth resolves
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
    })
  })

  it('shows error and does NOT navigate when reauthenticate fails after login', async () => {
    mockPost.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', name: 'Test' },
    })
    mockReauthenticate.mockResolvedValue(false)

    renderWithRouter(<Login />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'user@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // reauthenticate was called
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

    // Error message is displayed
    await waitFor(() => {
      expect(
        screen.getByText('Login succeeded but session could not be verified. Please try again.'),
      ).toBeInTheDocument()
    })

    // Navigation does NOT happen
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('does NOT call reauthenticate when login API returns an error', async () => {
    mockPost.mockResolvedValue({
      error: { code: 'invalid_credentials', message: 'Invalid email or password' },
    })

    renderWithRouter(<Login />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'bad@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'wrong' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // Error message is displayed
    await waitFor(() => {
      expect(screen.getByText('Invalid email or password')).toBeInTheDocument()
    })

    // reauthenticate is NOT called
    expect(mockReauthenticate).not.toHaveBeenCalled()
    // Navigation does NOT happen
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  // ── Ordering: navigation waits for reauthenticate resolution ────

  it('delays navigation until reauthenticate resolves, proving ordering', async () => {
    mockPost.mockResolvedValue({
      data: { id: '1', email: 'user@test.com', name: 'Test' },
    })

    // Create a deferred promise for reauthenticate so we can control when it resolves
    let resolveReauth!: (value: boolean) => void
    const reauthPromise = new Promise<boolean>((resolve) => {
      resolveReauth = resolve
    })
    mockReauthenticate.mockReturnValue(reauthPromise)

    renderWithRouter(<Login />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'user@test.com' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'pass123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    // reauthenticate is called after login succeeds
    await waitFor(() => {
      expect(mockReauthenticate).toHaveBeenCalledTimes(1)
    })

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
