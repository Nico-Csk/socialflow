import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '@/test/render'

// Mock apiClient BEFORE importing TaskForm
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
import TaskForm from '@/pages/Tasks/TaskForm'

describe('TaskForm — fetching-state behavior preservation (lint hardening)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useParams).mockReturnValue({})
  })

  it('in create mode, fetching=false, form renders immediately without Loading... text', async () => {
    vi.mocked(useParams).mockReturnValue({})

    renderWithRouter(<TaskForm />)

    // Heading must render immediately
    await waitFor(() => {
      expect(screen.getByText('New Task')).toBeInTheDocument()
    })

    // Loading text must NOT appear in create mode
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()

    // Form fields must be visible
    expect(screen.getByLabelText('Title *')).toBeInTheDocument()
    expect(screen.getByLabelText('Description')).toBeInTheDocument()

    // Create button label
    expect(screen.getByRole('button', { name: 'Create Task' })).toBeInTheDocument()
  })

  it('in edit mode, fetching starts as true, shows Loading..., then pre-fills form fields', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'task-789' })

    mockGet.mockResolvedValue({
      data: {
        title: 'Review Instagram draft',
        description: 'Check grammar and branding',
        assignee_id: null,
        due_date: '2026-06-20',
        content_item_id: null,
        client_id: null,
        done: false,
      },
    })

    renderWithRouter(<TaskForm />)

    // Loading... must appear while fetching
    expect(screen.getByText('Loading...')).toBeInTheDocument()

    // API call must fire with the correct endpoint
    expect(mockGet).toHaveBeenCalledWith('/tasks/task-789')

    // Wait for data and form to render
    await waitFor(() => {
      expect(screen.getByText('Edit Task')).toBeInTheDocument()
    })

    // Title field must be pre-filled
    const titleInput = screen.getByLabelText('Title *') as HTMLInputElement
    expect(titleInput.value).toBe('Review Instagram draft')

    // Due date must be pre-filled
    const dueInput = screen.getByLabelText('Due Date') as HTMLInputElement
    expect(dueInput.value).toBe('2026-06-20')

    // Done checkbox must appear in edit mode
    expect(screen.getByLabelText('Mark as done')).toBeInTheDocument()

    // Loading must be gone
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
  })

  it('in create mode, done checkbox is NOT rendered', async () => {
    vi.mocked(useParams).mockReturnValue({})

    renderWithRouter(<TaskForm />)

    await waitFor(() => {
      expect(screen.getByText('New Task')).toBeInTheDocument()
    })

    // Done checkbox should NOT exist in create mode
    expect(screen.queryByLabelText('Mark as done')).not.toBeInTheDocument()
  })

  it('in edit mode, shows error when API returns error', async () => {
    vi.mocked(useParams).mockReturnValue({ id: 'task-err' })

    mockGet.mockResolvedValue({
      error: { code: 'not_found', message: 'Task not found' },
    })

    renderWithRouter(<TaskForm />)

    await waitFor(() => {
      expect(screen.getByText('Task not found')).toBeInTheDocument()
    })
  })
})
