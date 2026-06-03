import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '@/test/render'

// Mock apiClient BEFORE importing ClientForm
const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()

vi.mock('@/lib/apiClient', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
  },
}))

// Mock useParams to control edit/create mode
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
  }
})

import { useParams } from 'react-router-dom'
import ClientForm from '@/pages/Clients/ClientForm'

describe('ClientForm — fetching-state behavior preservation (lint hardening)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Default: create mode (no id)
    vi.mocked(useParams).mockReturnValue({})
  })

  it('in create mode, fetching starts as false and form renders immediately without Loading... text', async () => {
    // create mode: useParams returns {} → isEdit = false
    vi.mocked(useParams).mockReturnValue({})

    renderWithRouter(<ClientForm />)

    // Form heading should render immediately (no "Loading..." shown)
    await waitFor(() => {
      expect(screen.getByText('New Client')).toBeInTheDocument()
    })

    // Loading text should NOT appear in create mode
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()

    // Form fields should be visible
    expect(screen.getByLabelText('Name *')).toBeInTheDocument()
    expect(screen.getByLabelText('Notes')).toBeInTheDocument()

    // Submit button should show create label
    expect(screen.getByRole('button', { name: 'Create Client' })).toBeInTheDocument()
  })

  it('in edit mode, fetching starts as true, shows Loading... then renders form with pre-filled data', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'client-123' })

    mockGet.mockResolvedValue({
      data: { name: 'Acme Corp', notes: 'Top client', active: true },
    })

    renderWithRouter(<ClientForm />)

    // Loading... must appear while fetching is true
    expect(screen.getByText('Loading...')).toBeInTheDocument()

    // API must be called with the correct endpoint
    expect(mockGet).toHaveBeenCalledWith('/clients/client-123')

    // Wait for data to load and form to render
    await waitFor(() => {
      expect(screen.getByText('Edit Client')).toBeInTheDocument()
    })

    // Form fields must be pre-filled with API data
    const nameInput = screen.getByLabelText('Name *') as HTMLInputElement
    expect(nameInput.value).toBe('Acme Corp')

    // Loading text must disappear after data resolves
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
  })

  it('in edit mode, shows error when API returns an error', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'client-err' })

    mockGet.mockResolvedValue({
      error: { code: 'not_found', message: 'Client not found' },
    })

    renderWithRouter(<ClientForm />)

    // Should call the correct endpoint
    expect(mockGet).toHaveBeenCalledWith('/clients/client-err')

    // Error should surface as text after loading resolves
    await waitFor(() => {
      expect(screen.getByText('Client not found')).toBeInTheDocument()
    })
  })

  it('submit button shows Save Changes in edit mode after data loads', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'client-456' })

    mockGet.mockResolvedValue({
      data: { name: 'Beta Inc', notes: '', active: false },
    })

    renderWithRouter(<ClientForm />)

    await waitFor(() => {
      expect(screen.getByText('Edit Client')).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: 'Save Changes' })).toBeInTheDocument()
  })
})
