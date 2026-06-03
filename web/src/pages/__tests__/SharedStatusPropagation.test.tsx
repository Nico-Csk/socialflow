import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithRouter } from '@/test/render'

// ── Shared mocks for apiClient ─────────────────────────────────────
const mockGet = vi.fn()
const mockPatch = vi.fn()

vi.mock('@/lib/apiClient', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    patch: (...args: unknown[]) => mockPatch(...args),
  },
}))

// ── Mock react-router-dom (only override useParams) ────────────────
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
  }
})

// ── THE KEY: augment the shared NEXT_STATUS map ────────────────────
// Adding 'rejected' to review transitions. Both Calendar and ContentDetail
// import from @/lib/statusTransitions, so this ONE mock propagates to both.
vi.mock('@/lib/statusTransitions', () => ({
  NEXT_STATUS: {
    draft: ['review'],
    review: ['draft', 'approved', 'rejected'], // ← NEW: 'rejected' added
    approved: ['published'],
    published: ['archived'],
    archived: [],
  },
}))

import Calendar from '@/pages/Calendar/Calendar'
import ContentDetail from '@/pages/ContentItems/ContentDetail'
import { useParams } from 'react-router-dom'

const dateStr = '2026-05-15'
const monthStr = '2026-05'

// ── status-transitions-shared S3.1: Calendar sees augmented map ────
describe('Shared NEXT_STATUS propagation — Calendar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useParams).mockReturnValue({})
  })

  it('renders "Rejected" transition button for a review item (added to shared map)', async () => {
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-r1',
            title: 'Review Item',
            platform: 'instagram',
            content_type: 'post',
            status: 'review',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 1 },
      },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${monthStr}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Original transitions still present
    expect(screen.getByRole('button', { name: 'Move to Draft' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Move to Approved' })).toBeInTheDocument()

    // NEW transition from the augmented shared map
    expect(screen.getByRole('button', { name: 'Move to Rejected' })).toBeInTheDocument()
  })
})

// ── status-transitions-shared S3.1: ContentDetail sees the SAME augmented map
describe('Shared NEXT_STATUS propagation — ContentDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useParams).mockReturnValue({ id: 'ci-r1' })
  })

  it('renders "Rejected" transition button for a review item (added to shared map)', async () => {
    mockGet.mockResolvedValue({
      data: {
        id: 'ci-r1',
        workspace_id: 'ws1',
        client_id: null,
        title: 'Review Item',
        description: '',
        platform: 'instagram',
        content_type: 'post',
        status: 'review',
        scheduled_date: null,
        created_by: 'user1',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
        comments: [],
      },
    })

    renderWithRouter(<ContentDetail />)

    await waitFor(() => {
      expect(screen.getByText('Review Item')).toBeInTheDocument()
    })

    // Original transitions still present
    expect(screen.getByRole('button', { name: 'Draft' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Approved' })).toBeInTheDocument()

    // NEW transition from the augmented shared map — ContentDetail buttons
    // use text content as accessible name (no aria-label), so just 'Rejected'
    expect(screen.getByRole('button', { name: 'Rejected' })).toBeInTheDocument()
  })
})
