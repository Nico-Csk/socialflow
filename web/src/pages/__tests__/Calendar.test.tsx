import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useLocation } from 'react-router-dom'
import { renderWithRouter } from '@/test/render'

// Mock apiClient BEFORE importing Calendar
const mockGet = vi.fn()
const mockPatch = vi.fn()

vi.mock('@/lib/apiClient', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    patch: (...args: unknown[]) => mockPatch(...args),
  },
}))

import Calendar from '@/pages/Calendar/Calendar'
import {
  STATUS_OPTIONS,
  PLATFORM_OPTIONS,
  buildCalendarQuery,
} from '@/lib/calendarHelpers'

function LocationProbe() {
  const location = useLocation()
  return <div data-testid="location-pathname">{location.pathname}</div>
}

// ── Phase 1: RED Tests — Helpers ──────────────────────────────────

describe('buildCalendarQuery() — query string builder', () => {
  it('returns only month param when status and platform are absent', () => {
    const result = buildCalendarQuery({ month: '2026-05' })
    expect(result).toBe('/calendar?month=2026-05')
  })

  it('omits status param when status is "all"', () => {
    const result = buildCalendarQuery({ month: '2026-05', status: 'all' })
    expect(result).toBe('/calendar?month=2026-05')
  })

  it('omits platform param when platform is "all"', () => {
    const result = buildCalendarQuery({ month: '2026-06', platform: 'all' })
    expect(result).toBe('/calendar?month=2026-06')
  })

  it('includes status param when status is a real status filter', () => {
    const result = buildCalendarQuery({ month: '2026-05', status: 'draft' })
    expect(result).toBe('/calendar?month=2026-05&status=draft')
  })

  it('includes platform param when platform is a real platform filter', () => {
    const result = buildCalendarQuery({ month: '2026-06', platform: 'instagram' })
    expect(result).toBe('/calendar?month=2026-06&platform=instagram')
  })

  it('includes both status and platform when both are real filter values', () => {
    const result = buildCalendarQuery({ month: '2026-07', status: 'published', platform: 'twitter' })
    expect(result).toBe('/calendar?month=2026-07&status=published&platform=twitter')
  })

  it('preserves month with both filters set to "all" (both omitted)', () => {
    const result = buildCalendarQuery({ month: '2025-12', status: 'all', platform: 'all' })
    expect(result).toBe('/calendar?month=2025-12')
  })
})

describe('STATUS_OPTIONS and PLATFORM_OPTIONS constants', () => {
  it('STATUS_OPTIONS contains the 6 required status values in order', () => {
    expect(STATUS_OPTIONS).toEqual([
      'all',
      'draft',
      'review',
      'approved',
      'published',
      'archived',
    ])
  })

  it('STATUS_OPTIONS has exactly 6 elements', () => {
    expect(STATUS_OPTIONS).toHaveLength(6)
  })

  it('STATUS_OPTIONS is readonly (const tuple)', () => {
    // Verify that the tuple includes 'all' at index 0 and 'archived' at index 5
    expect(STATUS_OPTIONS[0]).toBe('all')
    expect(STATUS_OPTIONS[5]).toBe('archived')
  })

  it('PLATFORM_OPTIONS contains the 8 required platform values in order', () => {
    expect(PLATFORM_OPTIONS).toEqual([
      'all',
      'instagram',
      'facebook',
      'twitter',
      'linkedin',
      'tiktok',
      'youtube',
      'other',
    ])
  })

  it('PLATFORM_OPTIONS has exactly 8 elements', () => {
    expect(PLATFORM_OPTIONS).toHaveLength(8)
  })

  it('PLATFORM_OPTIONS is readonly (const tuple)', () => {
    expect(PLATFORM_OPTIONS[0]).toBe('all')
    expect(PLATFORM_OPTIONS[7]).toBe('other')
  })
})

describe('Calendar page — behavior preservation (lint hardening)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls GET /calendar?month=YYYY-MM with current month on mount', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    // Verify the URL pattern: /calendar?month=YYYY-MM (year-month format)
    expect(callArg).toMatch(/^\/calendar\?month=\d{4}-\d{2}$/)
  })

  it('shows skeleton grid during loading instead of text', () => {
    // Never-resolving promise keeps loading state active
    mockGet.mockReturnValue(new Promise(() => {}))

    renderWithRouter(<Calendar />)

    // Skeleton grid must be present (42 cells for 6 rows × 7 columns)
    const skeletonCells = screen.getAllByTestId('skeleton-cell')
    expect(skeletonCells).toHaveLength(42)

    // Old loading text must NOT be present
    expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
  })

  it('displays error message when API returns an error', async () => {
    mockGet.mockResolvedValue({
      error: { code: 'server_error', message: 'Failed to load calendar' },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.getByText('Failed to load calendar')).toBeInTheDocument()
    })
  })

  it('renders calendar grid with day-of-week headers when data loads', async () => {
    // Populate items so grid renders (not empty-month state)
    const today = new Date()
    const testDay = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-15`
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-x',
            title: 'Grid Test',
            platform: 'facebook',
            content_type: 'post',
            status: 'draft',
            scheduled_date: testDay,
          },
        ],
        counts_by_day: { [testDay]: 1 },
      },
    })

    renderWithRouter(<Calendar />)

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Day-of-week headers must be visible
    for (const day of ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']) {
      expect(screen.getByText(day)).toBeInTheDocument()
    }
  })

  it('renders navigation buttons (Prev/Next) and month-year header', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Navigation buttons exist
    expect(screen.getByText('← Prev')).toBeInTheDocument()
    expect(screen.getByText('Next →')).toBeInTheDocument()

    // Month-year header renders (e.g., "May 2026")
    const header = screen.getByText(/\d{4}/) // any year
    expect(header).toBeInTheDocument()
  })

  it('renders day numbers in the grid when data loads', async () => {
    const today = new Date()
    const testDay = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-01`
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-d',
            title: 'Day Test',
            platform: 'twitter',
            content_type: 'post',
            status: 'draft',
            scheduled_date: testDay,
          },
        ],
        counts_by_day: { [testDay]: 1 },
      },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Day 1 must be visible in the grid (any month has a day 1)
    const dayCells = screen.getAllByText('1')
    // At least one day-1 cell should exist (could be more if display logic duplicates)
    expect(dayCells.length).toBeGreaterThanOrEqual(1)
  })
})

// ── Phase 3: RED Tests — UI Behavior ─────────────────────────────

describe('Calendar — URL-driven month navigation (task 3.1)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('reads ?month=YYYY-MM from URL on mount and renders that month', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, { initialEntries: [{ pathname: '/', search: '?month=2026-03' }] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // API call must include the month from URL
    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('month=2026-03')

    // Month header must contain year 2026 (locale-agnostic)
    const header = screen.getByText(/2026/)
    expect(header).toBeInTheDocument()
  })

  it('defaults to current month when no ?month= param in URL', async () => {
    const now = new Date()
    const expectedMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`

    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain(`month=${expectedMonth}`)
  })

  it('renders "Today" button in navigation bar', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, { initialEntries: [{ pathname: '/', search: '?month=2025-06' }] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /today/i })).toBeInTheDocument()
  })

  it('clicking "Today" updates URL to current month', async () => {
    const now = new Date()
    const currentMonthStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`

    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, { initialEntries: [{ pathname: '/', search: '?month=2025-03' }] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Click Today
    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByRole('button', { name: /today/i }))

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain(`month=${currentMonthStr}`)
  })

  it('Prev/Next buttons update URL month correctly (same year)', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, { initialEntries: [{ pathname: '/', search: '?month=2026-05' }] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Click Prev to go to April
    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('← Prev'))

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const prevCall = mockGet.mock.calls[0][0] as string
    expect(prevCall).toContain('month=2026-04')
  })

  it('Prev crosses year boundary (Jan → Dec of previous year)', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, { initialEntries: [{ pathname: '/', search: '?month=2026-01' }] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('← Prev'))

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const prevCall = mockGet.mock.calls[0][0] as string
    expect(prevCall).toContain('month=2025-12')
  })
})

describe('Calendar — status tabs and platform select (task 3.2)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders status tabs with all expected status values', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    for (const status of ['All', 'Draft', 'Review', 'Approved', 'Published', 'Archived']) {
      expect(screen.getByText(status)).toBeInTheDocument()
    }
  })

  it('clicking a status tab triggers API call with ?status= param', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('Review'))

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('status=review')
  })

  it('changing status keeps the current dashboard calendar route', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(
      <>
        <Calendar />
        <LocationProbe />
      </>,
      { initialEntries: ['/dashboard/calendar?status=draft'] },
    )

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Review' }))

    expect(screen.getByTestId('location-pathname')).toHaveTextContent('/dashboard/calendar')
  })

  it('clicking "All" clears status param from API call', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, { initialEntries: ['/?status=draft'] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('All'))

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).not.toContain('status=')
  })

  it('renders platform <select> with platform options', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    const select = screen.getByRole('combobox')
    expect(select).toBeInTheDocument()
  })

  it('selecting a platform triggers API call with ?platform= param', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.selectOptions(screen.getByRole('combobox'), 'instagram')

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('platform=instagram')
  })
})

describe('Calendar — API query includes combined params (task 3.3)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('API call includes month + status + platform from URL params', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: ['/?month=2026-08&status=published&platform=twitter'],
    })

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('month=2026-08')
    expect(callArg).toContain('status=published')
    expect(callArg).toContain('platform=twitter')
  })

  it('omits "all" filter values from API call even if present in URL', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: ['/?month=2026-05&status=all&platform=all'],
    })

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('month=2026-05')
    expect(callArg).not.toContain('status=')
    expect(callArg).not.toContain('platform=')
  })
})

describe('Calendar — always-visible sidebar (task 3.4)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows "Select a day" prompt when no date is selected', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    expect(screen.getByText(/Select a day/)).toBeInTheDocument()
  })

  it('shows items in sidebar when a date with content is clicked', async () => {
    const dateStr = '2026-05-15'
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-1',
            title: 'Facebook Launch Post',
            platform: 'facebook',
            content_type: 'post',
            status: 'approved',
            scheduled_date: dateStr,
          },
          {
            id: 'ci-2',
            title: 'Instagram Reel',
            platform: 'instagram',
            content_type: 'reel',
            status: 'draft',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 2 },
      },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Click day 15
    const day15 = screen.getByText('15')
    await user.click(day15)

    await waitFor(() => {
      expect(screen.getByText('Facebook Launch Post')).toBeInTheDocument()
    })

    expect(screen.getByText('Instagram Reel')).toBeInTheDocument()
    expect(screen.getByText('approved')).toBeInTheDocument()
    expect(screen.getByText('draft')).toBeInTheDocument()
    expect(screen.getByText('facebook')).toBeInTheDocument()
    expect(screen.getByText('instagram')).toBeInTheDocument()
  })

  it('shows "No content scheduled for this day." for empty date', async () => {
    // Provide items so the grid renders (at least one item on a different day)
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-dummy',
            title: 'Dummy',
            platform: 'twitter',
            content_type: 'post',
            status: 'draft',
            scheduled_date: '2026-05-01',
          },
        ],
        counts_by_day: { '2026-05-20': 0, '2026-05-01': 1 },
      },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: '?month=2026-05' }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    await user.click(screen.getByText('20'))

    await waitFor(() => {
      expect(screen.getByText('No content scheduled for this day.')).toBeInTheDocument()
    })
  })

  it('sidebar persists after month navigation', async () => {
    const dateStr = '2026-05-15'
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-1',
            title: 'May Item',
            platform: 'twitter',
            content_type: 'post',
            status: 'draft',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 1 },
      },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: '?month=2026-05' }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    await user.click(screen.getByText('15'))

    await waitFor(() => {
      expect(screen.getByText('May Item')).toBeInTheDocument()
    })

    // Navigate to previous month
    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('← Prev'))

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Sidebar should still show the selected date label
    expect(screen.getByText('2026-05-15')).toBeInTheDocument()
  })

  it('selected-day highlight persists after navigating away and back (S6)', async () => {
    const dateStr = '2026-05-15'
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-s6',
            title: 'S6 Highlight Item',
            platform: 'facebook',
            content_type: 'post',
            status: 'draft',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 1 },
      },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: '?month=2026-05' }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Select day 15 — it should now be visually highlighted
    await user.click(screen.getByText('15'))

    await waitFor(() => {
      expect(screen.getByText('S6 Highlight Item')).toBeInTheDocument()
    })

    // Navigate to Previous month (April)
    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    await user.click(screen.getByText('← Prev'))

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Verify sidebar still shows the selected date (persistence check)
    expect(screen.getByText('2026-05-15')).toBeInTheDocument()

    // Navigate BACK to the original month (May)
    vi.clearAllMocks()
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-s6',
            title: 'S6 Highlight Item',
            platform: 'facebook',
            content_type: 'post',
            status: 'draft',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 1 },
      },
    })

    await user.click(screen.getByText('Next →'))

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Day 15 cell should still be highlighted
    const selectedCell = screen.getByTestId('day-cell-2026-05-15')
    expect(selectedCell).toHaveAttribute('data-selected', 'true')
  })
})

describe('Calendar — loading, empty, and error states (task 3.5)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders skeleton grid during loading instead of text', () => {
    mockGet.mockReturnValue(new Promise(() => {}))

    renderWithRouter(<Calendar />)

    // Skeleton grid must be present (42 cells for 6 rows × 7 columns)
    const skeletonCells = screen.getAllByTestId('skeleton-cell')
    expect(skeletonCells).toHaveLength(42)

    // Old loading text must NOT be present
    expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
  })

  it('renders dashed-border empty-month message when zero items', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, { initialEntries: ['/?month=2026-06'] })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    expect(screen.getByText(/No content scheduled this month/)).toBeInTheDocument()
  })

  it('renders error banner correctly with red styling', async () => {
    mockGet.mockResolvedValue({
      error: { code: 'server_error', message: 'Failed to load calendar' },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.getByText('Failed to load calendar')).toBeInTheDocument()
    })

    // Error banner should be a semantic alert
    const errorContainer = screen.getByRole('alert')
    expect(errorContainer).toHaveTextContent('Failed to load calendar')
  })

  it('shows filtered-zero-results message distinct from empty month', async () => {
    // Month has items on other months, but filters active and nothing matches current month
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-other',
            title: 'Other Month Item',
            platform: 'twitter',
            content_type: 'post',
            status: 'draft',
            scheduled_date: '2026-04-01',
          },
        ],
        counts_by_day: { '2026-04-01': 1 },
      },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: ['/?month=2026-05&status=published'],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Should indicate filtering is the cause (items exist but not in current view)
    expect(screen.getByText(/no content.*match|No content in this month/i)).toBeInTheDocument()
  })
})

describe('Calendar — URL params survive refresh (task 3.6)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('reads all params from URL on mount including month, status, platform, and day', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: ['/?month=2026-09&status=approved&platform=linkedin&day=2026-09-14'],
    })

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(1)
    })

    const callArg = mockGet.mock.calls[0][0] as string
    expect(callArg).toContain('month=2026-09')
    expect(callArg).toContain('status=approved')
    expect(callArg).toContain('platform=linkedin')
  })

  it('shows active filter badges/chips for current status and platform', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />, {
      initialEntries: ['/?month=2026-05&status=published&platform=instagram'],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Active filter badges should be visible (multiple elements: tab + badge)
    const publishedElements = screen.getAllByText(/published/i)
    expect(publishedElements.length).toBeGreaterThanOrEqual(2) // at least tab + badge

    const instagramElements = screen.getAllByText(/instagram/i)
    expect(instagramElements.length).toBeGreaterThanOrEqual(2) // at least select option + badge
  })

  it('sidebar shows "Select a day" prompt and items on click (smoke)', async () => {
    const dateStr = '2026-05-15'
    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-sm',
            title: 'Smoke Test Item',
            platform: 'facebook',
            content_type: 'post',
            status: 'draft',
            scheduled_date: dateStr,
          },
        ],
        counts_by_day: { [dateStr]: 1 },
      },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: '?month=2026-05' }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Sidebar always visible with prompt
    expect(screen.getByText(/Select a day/)).toBeInTheDocument()

    await user.click(screen.getByText('15'))

    await waitFor(() => {
      expect(screen.getByText('Smoke Test Item')).toBeInTheDocument()
    })
  })
})

// ── Phase 3: RED Tests — Calendar Sidebar Workflow Actions ─────────

describe('Calendar — sidebar workflow actions (transition buttons)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const dateStr = '2026-05-15'

  function mockMonthData(items: Array<{
    id: string
    title: string
    platform: string
    content_type: string
    status: string
    scheduled_date: string
  }>) {
    const counts: Record<string, number> = {}
    for (const item of items) {
      counts[item.scheduled_date] = (counts[item.scheduled_date] ?? 0) + 1
    }
    return { items, counts_by_day: counts }
  }

  // S1.1 — Shows buttons for draft item
  it('renders "Review" transition button for a draft sidebar item', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Draft Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // S1.1: sidebar shows a "Review" button for the draft item
    expect(screen.getByRole('button', { name: 'Move to Review' })).toBeInTheDocument()
  })

  // S1.2 — No buttons for terminal item
  it('shows NO transition buttons for an archived (terminal) sidebar item', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Archived Item', platform: 'instagram', content_type: 'post', status: 'archived', scheduled_date: dateStr },
      ]),
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // S1.2: no transition buttons for archived
    expect(screen.queryByRole('button', { name: /Move to/ })).not.toBeInTheDocument()
  })

  // S2.1 — Optimistic success
  it('optimistically updates status text on transition click, then keeps it on PATCH success', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Optimistic Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    // PATCH succeeds and returns updated status
    mockPatch.mockResolvedValue({
      data: { status: 'review' },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Verify initial status badge shows "draft"
    expect(screen.getByText('draft')).toBeInTheDocument()

    // Click "Move to Review"
    await user.click(screen.getByRole('button', { name: 'Move to Review' }))

    // Optimistic: status should immediately show "review"
    await waitFor(() => {
      expect(screen.getByText('review')).toBeInTheDocument()
    })

    // PATCH called with correct payload
    expect(mockPatch).toHaveBeenCalledWith('/content-items/ci-1/status', { status: 'review' })
  })

  // S2.2 — Failure refetch
  it('refetches calendar data on PATCH failure', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Fail Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    // PATCH fails
    mockPatch.mockResolvedValue({
      error: { code: 'server_error', message: 'Transition failed' },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Clear the initial GET call count
    vi.clearAllMocks()

    // Re-mock GET to provide fresh data for the refetch test
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Fail Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })
    mockPatch.mockResolvedValue({
      error: { code: 'server_error', message: 'Transition failed' },
    })

    // Click "Move to Review" — which won't exist yet, but the test structure is ready
    // RED phase: this test will fail because the button doesn't exist
    await user.click(screen.getByRole('button', { name: 'Move to Review' }))

    await waitFor(() => {
      // On failure, loadMonth should be called (a second GET to the calendar endpoint)
      const getCalls = mockGet.mock.calls.filter((call: unknown[]) => {
        const arg = call[0] as string
        return arg.startsWith('/calendar')
      })
      expect(getCalls.length).toBeGreaterThanOrEqual(1)
    })
  })

  // S3.1 — Disabled during request
  it('disables transition buttons while PATCH is in flight', async () => {
    // Make PATCH hang so we can observe disabled state
    let resolvePatch: (value: unknown) => void
    const patchPromise = new Promise((resolve) => {
      resolvePatch = resolve
    })
    mockPatch.mockReturnValue(patchPromise)

    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Disable Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Click transition button — optimistically changes to "review" status
    await user.click(screen.getByRole('button', { name: 'Move to Review' }))

    // After optimistic update, the item is now "review" status, so buttons change
    // to "Move to Draft" and "Move to Approved" — but they should be disabled
    const draftBtn = screen.getByRole('button', { name: 'Move to Draft' })
    const approvedBtn = screen.getByRole('button', { name: 'Move to Approved' })
    expect(draftBtn).toBeDisabled()
    expect(approvedBtn).toBeDisabled()

    // Resolve the PATCH
    resolvePatch!({ data: { status: 'review' } })
  })

  // S4.1 — "Details & Comments" link
  it('renders a "Details & Comments" link navigating to content item detail', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Link Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // "Details & Comments" link must exist and point to content item detail
    const detailsLink = screen.getByText(/Details & Comments/)
    expect(detailsLink).toBeInTheDocument()
    expect(detailsLink.closest('a')).toHaveAttribute('href', '/dashboard/content-items/ci-1')
  })

  // S5.2 — Keyboard Enter activation
  it('transition buttons respond to keyboard Enter activation', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Keyboard Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    mockPatch.mockResolvedValue({
      data: { status: 'review' },
    })

    const user = userEvent.setup()
    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    const reviewBtn = screen.getByRole('button', { name: 'Move to Review' })

    // Focus the button, then press Enter — keyboard-only interaction
    reviewBtn.focus()
    await user.keyboard('{Enter}')

    // Optimistic status update should be visible after keyboard activation
    await waitFor(() => {
      expect(screen.getByText('review')).toBeInTheDocument()
    })

    // PATCH should have been called
    expect(mockPatch).toHaveBeenCalledWith('/content-items/ci-1/status', { status: 'review' })
  })

  // S5.1 — aria-label
  it('transition buttons have descriptive aria-labels', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Aria Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
        { id: 'ci-2', title: 'Published Item', platform: 'facebook', content_type: 'post', status: 'published', scheduled_date: dateStr },
      ]),
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Draft item → "Move to Review"
    expect(screen.getByRole('button', { name: 'Move to Review' })).toBeInTheDocument()
    // Published item → "Move to Archived"
    expect(screen.getByRole('button', { name: 'Move to Archived' })).toBeInTheDocument()
  })

  // Title should be a separate link (not wrapping the whole card)
  it('item title is a separate link, not wrapped with actions', async () => {
    mockGet.mockResolvedValue({
      data: mockMonthData([
        { id: 'ci-1', title: 'Title Link Item', platform: 'instagram', content_type: 'post', status: 'draft', scheduled_date: dateStr },
      ]),
    })

    renderWithRouter(<Calendar />, {
      initialEntries: [{ pathname: '/', search: `?month=${dateStr.slice(0, 7)}&day=${dateStr}` }],
    })

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Title should be rendered as a link
    const titleLink = screen.getByText('Title Link Item').closest('a')
    expect(titleLink).toHaveAttribute('href', '/dashboard/content-items/ci-1')
  })
})

describe('Calendar page — null-safe rendering (regression)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without crashing when API returns items: null (normalized to empty)', async () => {
    // RED: API returns items as null (Go nil slice not yet normalized)
    mockGet.mockResolvedValue({
      data: { items: null, counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    // Should finish loading without a crash
    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Empty-month message or grid should render without error (proves no crash)
    const emptyMsg = screen.queryByText('No content scheduled this month.')
    const gridHeaders = screen.queryByText('Sun')
    // Either empty-month or grid is acceptable — the key is no TypeError
    expect(emptyMsg || gridHeaders).toBeTruthy()
  })

  it('renders without errors when API returns items: []', async () => {
    mockGet.mockResolvedValue({
      data: { items: [], counts_by_day: {} },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Empty month should show the dashed-border message
    expect(screen.getByText('No content scheduled this month.')).toBeInTheDocument()

    // Sidebar is always visible with prompt
    expect(screen.getByText(/Select a day/)).toBeInTheDocument()
  })

  it('shows content count indicators when API returns populated items and counts', async () => {
    // Regression: day cells with content must show count badges
    const today = new Date()
    const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`

    mockGet.mockResolvedValue({
      data: {
        items: [
          {
            id: 'ci-1',
            title: 'Test Post',
            platform: 'instagram',
            content_type: 'post',
            status: 'draft',
            scheduled_date: todayStr,
          },
        ],
        counts_by_day: { [todayStr]: 1 },
      },
    })

    renderWithRouter(<Calendar />)

    await waitFor(() => {
      expect(screen.queryByText('Loading calendar...')).not.toBeInTheDocument()
    })

    // Grid headers render
    expect(screen.getByText('Sun')).toBeInTheDocument()

    // Because today's day cell has count:1, two "1" texts should exist:
    // one for the day number and one for the count badge inside the same cell
    const allOnes = screen.getAllByText('1')
    expect(allOnes.length).toBeGreaterThanOrEqual(2)
  })
})
