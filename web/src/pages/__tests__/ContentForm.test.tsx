import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '@/test/render'

// Mock apiClient BEFORE importing ContentForm
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
import ContentForm from '@/pages/ContentItems/ContentForm'

describe('ContentForm — fetching-state behavior preservation (lint hardening)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useParams).mockReturnValue({})
  })

  it('in create mode, fetching=false, form renders immediately, and clients preload fires', async () => {
    vi.mocked(useParams).mockReturnValue({})

    // Clients preload always fires; content item fetch only in edit mode
    mockGet.mockResolvedValue({
      data: [{ id: 'c1', name: 'Client A' }],
    })

    renderWithRouter(<ContentForm />)

    // Heading renders immediately in create mode (no Loading... shown)
    await waitFor(() => {
      expect(screen.getByText('New Content Item')).toBeInTheDocument()
    })

    // Loading text must NOT appear in create mode
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()

    // Clients preload API call must fire: GET /clients
    expect(mockGet).toHaveBeenCalledWith('/clients')

    // Form fields must be visible
    expect(screen.getByLabelText('Title *')).toBeInTheDocument()
    expect(screen.getByLabelText('Description')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create Item' })).toBeInTheDocument()
  })

  it('in edit mode, fetching starts as true, shows Loading..., then pre-fills form fields', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'content-456' })

    // Mock clients list AND content item data (both API calls in useEffect)
    mockGet
      .mockResolvedValueOnce({
        data: [{ id: 'c1', name: 'Client A' }],
      })
      .mockResolvedValueOnce({
        data: {
          title: 'Summer Campaign',
          description: 'A sunny campaign',
          platform: 'instagram',
          content_type: 'post',
          client_id: 'c1',
          scheduled_date: '2026-06-15',
        },
      })

    renderWithRouter(<ContentForm />)

    // Loading... must show while fetching is true
    expect(screen.getByText('Loading...')).toBeInTheDocument()

    // Both API calls must fire
    expect(mockGet).toHaveBeenCalledWith('/clients')
    expect(mockGet).toHaveBeenCalledWith('/content-items/content-456')

    // Wait for data to resolve and form to render
    await waitFor(() => {
      expect(screen.getByText('Edit Content Item')).toBeInTheDocument()
    })

    // Title field must be pre-filled
    const titleInput = screen.getByLabelText('Title *') as HTMLInputElement
    expect(titleInput.value).toBe('Summer Campaign')

    // Loading text must be gone
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
  })

  it('in create mode, clients preload still fires to populate client dropdown', async () => {
    vi.mocked(useParams).mockReturnValue({})

    mockGet.mockResolvedValue({
      data: [
        { id: 'c1', name: 'Client One' },
        { id: 'c2', name: 'Client Two' },
      ],
    })

    renderWithRouter(<ContentForm />)

    // Wait for clients preload to resolve and dropdown to populate
    await waitFor(() => {
      expect(screen.getByText('Client One')).toBeInTheDocument()
    })

    expect(screen.getByText('Client Two')).toBeInTheDocument()

    // Verify only /clients was called (no content item fetch)
    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/clients')
  })

  it('in edit mode, shows error when content item API returns error', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'content-err' })

    mockGet
      .mockResolvedValueOnce({ data: [] }) // clients preload
      .mockResolvedValueOnce({
        error: { code: 'not_found', message: 'Content item not found' },
      })

    renderWithRouter(<ContentForm />)

    await waitFor(() => {
      expect(screen.getByText('Content item not found')).toBeInTheDocument()
    })
  })
})
