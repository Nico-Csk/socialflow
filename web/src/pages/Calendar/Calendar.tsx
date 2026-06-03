import { useState, useEffect, useCallback } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import apiClient from '@/lib/apiClient'
import { STATUS_OPTIONS, PLATFORM_OPTIONS, buildCalendarQuery } from '@/lib/calendarHelpers'
import { NEXT_STATUS } from '@/lib/statusTransitions'

const STATUS_LABELS: Record<string, string> = {
  all: 'All',
  draft: 'Draft',
  review: 'Review',
  approved: 'Approved',
  published: 'Published',
  archived: 'Archived',
}

const PLATFORM_LABELS: Record<string, string> = {
  all: 'All platforms',
  instagram: 'Instagram',
  facebook: 'Facebook',
  twitter: 'Twitter',
  linkedin: 'LinkedIn',
  tiktok: 'TikTok',
  youtube: 'YouTube',
  other: 'Other',
}

// ── Helpers ─────────────────────────────────────────────────────────
function getCurrentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function monthToParts(monthStr: string): { year: number; month: number } {
  const [y, m] = monthStr.split('-').map(Number)
  return { year: y, month: m }
}

function partsToMonth(year: number, month: number): string {
  return `${year}-${String(month).padStart(2, '0')}`
}

// ── Types ───────────────────────────────────────────────────────────
interface ContentItem {
  id: string
  title: string
  platform: string
  content_type: string
  status: string
  scheduled_date: string | null
}

interface CalendarResult {
  items: ContentItem[]
  counts_by_day: Record<string, number>
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  review: 'bg-yellow-100 text-yellow-800',
  approved: 'bg-blue-100 text-blue-800',
  published: 'bg-green-100 text-green-800',
  archived: 'bg-red-100 text-red-800',
}

// ── Skeleton Grid ───────────────────────────────────────────────────
function SkeletonGrid() {
  const cells = Array.from({ length: 42 }) // 6 rows × 7 columns
  return (
    <div className="flex-1 bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="grid grid-cols-7 bg-gray-50 border-b border-gray-200">
        {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((d) => (
          <div key={d} className="px-2 py-2 text-center text-xs font-medium text-gray-500">
            {d}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7">
        {cells.map((_, i) => (
          <div
            key={i}
            className="min-h-[80px] border-b border-r border-gray-100 p-1.5 animate-pulse"
            data-testid="skeleton-cell"
          >
            <div className="w-6 h-6 rounded-full bg-gray-200 mb-1" />
            <div className="w-8 h-3 rounded bg-gray-100" />
          </div>
        ))}
      </div>
    </div>
  )
}

// ── Main Component ───────────────────────────────────────────────────
export default function Calendar() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [data, setData] = useState<CalendarResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [pendingTransitions, setPendingTransitions] = useState<Record<string, string>>({})

  // Read params from URL
  const month = searchParams.get('month') || getCurrentMonth()
  const status = searchParams.get('status') ?? ''
  const platform = searchParams.get('platform') ?? ''
  const day = searchParams.get('day') ?? ''

  // Active day — always show if day param exists (sidebar persists across month nav per spec S5/S6)
  const activeDay = day

  const hasActiveFilters = (status && status !== 'all') || (platform && platform !== 'all')

  const { year, month: monthNum } = monthToParts(month)
  const today = new Date()
  const todayStr = today.toISOString().slice(0, 10)
  const currentMonthStr = getCurrentMonth()

  const monthName = new Date(year, monthNum - 1).toLocaleString('default', { month: 'long' })

  const loadMonth = useCallback(async () => {
    setLoading(true)
    setError('')
    const queryPath = buildCalendarQuery({ month, status: status || undefined, platform: platform || undefined })
    const res = await apiClient.get<CalendarResult>(queryPath)
    if (res.error) {
      setError(res.error.message)
    } else if (res.data) {
      setData({
        ...res.data,
        items: res.data.items ?? [],
        counts_by_day: res.data.counts_by_day ?? {},
      })
    } else {
      setData(null)
    }
    setLoading(false)
  }, [month, status, platform])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadMonth()
  }, [loadMonth])

  // ── Sidebar transition handler ──────────────────────────────────

  async function handleSidebarTransition(itemId: string, nextStatus: string) {
    // Optimistic local update
    setData((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        items: prev.items.map((item) =>
          item.id === itemId ? { ...item, status: nextStatus } : item,
        ),
      }
    })

    // Mark as pending to disable buttons
    setPendingTransitions((prev) => ({ ...prev, [itemId]: nextStatus }))

    const res = await apiClient.patch<ContentItem>(`/content-items/${itemId}/status`, { status: nextStatus })

    // Clear pending regardless of outcome
    setPendingTransitions((prev) => {
      const next = { ...prev }
      delete next[itemId]
      return next
    })

    if (res.error) {
      setError(res.error.message)
      // Refetch full month data to restore canonical state
      await loadMonth()
    } else if (res.data) {
      // Update with server response to ensure consistency
      setData((prev) => {
        if (!prev) return prev
        return {
          ...prev,
          items: prev.items.map((item) =>
            item.id === itemId ? { ...item, status: res.data!.status } : item,
          ),
        }
      })
    }
  }

  // ── Navigation helpers ──────────────────────────────────────────

  function navToMonth(newMonth: string) {
    const newParams = new URLSearchParams(searchParams)
    newParams.set('month', newMonth)
    // Keep day param — sidebar persists across month navigation (spec S5/S6)
    setSearchParams(newParams, { replace: true })
  }

  function prevMonth() {
    if (monthNum === 1) {
      navToMonth(partsToMonth(year - 1, 12))
    } else {
      navToMonth(partsToMonth(year, monthNum - 1))
    }
  }

  function nextMonth() {
    if (monthNum === 12) {
      navToMonth(partsToMonth(year + 1, 1))
    } else {
      navToMonth(partsToMonth(year, monthNum + 1))
    }
  }

  function goToToday() {
    const newParams = new URLSearchParams(searchParams)
    newParams.set('month', currentMonthStr)
    setSearchParams(newParams, { replace: true })
  }

  // ── Filter helpers ───────────────────────────────────────────────

  function setPlatformFilter(value: string) {
    const newParams = new URLSearchParams(searchParams)
    if (value === 'all') {
      newParams.delete('platform')
    } else {
      newParams.set('platform', value)
    }
    setSearchParams(newParams, { replace: true })
  }

  function setStatusFilter(value: string) {
    const newParams = new URLSearchParams(searchParams)
    if (value === 'all') {
      newParams.delete('status')
    } else {
      newParams.set('status', value)
    }
    setSearchParams(newParams, { replace: true })
  }

  function selectDay(dateStr: string) {
    const newParams = new URLSearchParams(searchParams)
    newParams.set('day', dateStr)
    setSearchParams(newParams, { replace: true })
  }

  // ── Build calendar grid ──────────────────────────────────────────

  const firstDay = new Date(year, monthNum - 1, 1)
  const lastDay = new Date(year, monthNum, 0)
  const startDayOfWeek = firstDay.getDay() // 0=Sun
  const totalDays = lastDay.getDate()

  const days: (number | null)[] = []
  for (let i = 0; i < startDayOfWeek; i++) days.push(null)
  for (let d = 1; d <= totalDays; d++) days.push(d)

  const selectedItems = activeDay && data
    ? data.items.filter((item) => item.scheduled_date === activeDay)
    : []

  const isEmptyMonth = !loading && !error && data && (!data.items || data.items.length === 0)
  const isFilteredEmpty = !loading && !error && data && data.items && data.items.length > 0 &&
    hasActiveFilters && days.every((d) => {
      if (!d) return true
      const ds = `${year}-${String(monthNum).padStart(2, '0')}-${String(d).padStart(2, '0')}`
      return !data?.items.some((item) => item.scheduled_date === ds)
    })

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-gray-900">Calendar</h2>
        <div className="flex items-center gap-3">
          <button
            onClick={prevMonth}
            className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
          >
            ← Prev
          </button>
          <span className="text-lg font-medium text-gray-900 min-w-[160px] text-center">
            {monthName} {year}
          </span>
          <button
            onClick={nextMonth}
            className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
          >
            Next →
          </button>
          <button
            onClick={goToToday}
            className="rounded-lg border border-socialflow-300 px-3 py-1.5 text-sm font-medium text-socialflow-700 hover:bg-socialflow-50 transition-colors"
          >
            Today
          </button>
        </div>
      </div>

      {/* Filter controls */}
      <div className="flex items-center justify-between mb-3">
        {/* Status tabs */}
        <div className="flex gap-1">
          {STATUS_OPTIONS.map((s) => {
            const active = (s === 'all' && !status) || status === s
            return (
              <button
                type="button"
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                  active
                    ? 'bg-socialflow-100 text-socialflow-700'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}
              >
                {STATUS_LABELS[s]}
              </button>
            )
          })}
        </div>

        {/* Platform select */}
        <div className="flex items-center gap-2">
          <select
            value={platform || 'all'}
            onChange={(e) => setPlatformFilter(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-700 bg-white focus:outline-none focus:ring-2 focus:ring-socialflow-300 focus:border-socialflow-400"
          >
            {PLATFORM_OPTIONS.map((p) => (
              <option key={p} value={p}>
                {PLATFORM_LABELS[p]}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Active filter badges */}
      {hasActiveFilters && (
        <div className="flex items-center gap-2 mb-3">
          <span className="text-xs text-gray-500">Active filters:</span>
          {status && status !== 'all' && (
            <span className="inline-flex items-center gap-1 rounded-full bg-socialflow-100 px-2 py-0.5 text-xs font-medium text-socialflow-700">
              {STATUS_LABELS[status]}
            </span>
          )}
          {platform && platform !== 'all' && (
            <span className="inline-flex items-center gap-1 rounded-full bg-socialflow-100 px-2 py-0.5 text-xs font-medium text-socialflow-700">
              {PLATFORM_LABELS[platform]}
            </span>
          )}
        </div>
      )}

      {/* Error banner */}
      {error && (
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-4">
          {error}
        </div>
      )}

      {/* Loading skeleton */}
      {loading ? (
        <div className="flex gap-6">
          <SkeletonGrid />
          <div className="w-72 flex-shrink-0">
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="animate-pulse space-y-2">
                <div className="h-4 w-24 bg-gray-200 rounded" />
                <div className="h-3 w-32 bg-gray-100 rounded" />
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex gap-6">
          {/* Calendar grid */}
          {isEmptyMonth ? (
            <div className="flex-1 bg-white rounded-lg border border-dashed border-gray-300 p-12 text-center">
              <p className="text-gray-500">No content scheduled this month.</p>
            </div>
          ) : isFilteredEmpty ? (
            <div className="flex-1 bg-white rounded-lg border border-dashed border-gray-300 p-12 text-center">
              <p className="text-gray-500">
                No content in this month matches the current filters.
              </p>
            </div>
          ) : (
            <div className="flex-1 bg-white rounded-lg border border-gray-200 overflow-hidden">
              {/* Day-of-week headers */}
              <div className="grid grid-cols-7 bg-gray-50 border-b border-gray-200">
                {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((d) => (
                  <div key={d} className="px-2 py-2 text-center text-xs font-medium text-gray-500">
                    {d}
                  </div>
                ))}
              </div>

              {/* Day cells */}
              <div className="grid grid-cols-7">
                {days.map((dayNum, i) => {
                  const dateStr = dayNum
                    ? `${year}-${String(monthNum).padStart(2, '0')}-${String(dayNum).padStart(2, '0')}`
                    : null
                  const count = dateStr && data ? (data.counts_by_day[dateStr] ?? 0) : 0
                  const isToday = dateStr === todayStr
                  const isSelected = dateStr === activeDay

                  return (
                    <div
                      key={i}
                      data-testid={dateStr ? `day-cell-${dateStr}` : 'day-cell-empty'}
                      data-selected={isSelected ? 'true' : 'false'}
                      className={`min-h-[80px] border-b border-r border-gray-100 p-1.5 ${
                        dayNum ? 'cursor-pointer hover:bg-gray-50' : ''
                      } ${isSelected ? 'bg-socialflow-50 ring-1 ring-inset ring-socialflow-300' : ''}`}
                      onClick={() => dayNum && selectDay(dateStr!)}
                    >
                      {dayNum && (
                        <>
                          <span
                            className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium ${
                              isToday
                                ? 'bg-socialflow-600 text-white'
                                : 'text-gray-600'
                            }`}
                          >
                            {dayNum}
                          </span>
                          {count > 0 && (
                            <div className="mt-1">
                              <span className="inline-flex items-center rounded-full bg-socialflow-100 px-1.5 py-0.5 text-[10px] font-medium text-socialflow-700">
                                {count}
                              </span>
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Selected date sidebar — always visible */}
          <div className="w-72 flex-shrink-0">
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              {activeDay ? (
                <>
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">
                    {activeDay}
                  </h3>
                  {selectedItems.length === 0 ? (
                    <p className="text-xs text-gray-400">No content scheduled for this day.</p>
                  ) : (
                    <div className="space-y-2">
                      {selectedItems.map((item) => {
                        const isPending = pendingTransitions[item.id] !== undefined
                        const allowedNext = NEXT_STATUS[item.status] ?? []
                        return (
                          <div
                            key={item.id}
                            className="rounded-lg border border-gray-100 px-3 py-2"
                          >
                            {/* Title as separate link */}
                            <Link
                              to={`/dashboard/content-items/${item.id}`}
                              className="text-sm font-medium text-gray-900 truncate hover:text-socialflow-600 block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-socialflow-300 focus-visible:ring-offset-1 rounded"
                            >
                              {item.title}
                            </Link>

                            {/* Status badge and platform */}
                            <div className="flex gap-2 mt-1">
                              <span className={`inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium ${STATUS_COLORS[item.status] ?? 'bg-gray-100 text-gray-700'}`}>
                                {item.status}
                              </span>
                              <span className="text-[10px] text-gray-400 capitalize">{item.platform}</span>
                            </div>

                            {/* Inline transition buttons */}
                            {allowedNext.length > 0 && !isPending && (
                              <div className="flex flex-wrap gap-1 mt-2">
                                {allowedNext.map((next) => (
                                  <button
                                    key={next}
                                    onClick={() => handleSidebarTransition(item.id, next)}
                                    disabled={isPending}
                                    aria-label={`Move to ${next.charAt(0).toUpperCase() + next.slice(1)}`}
                                    className="rounded px-2 py-0.5 text-[10px] font-medium border border-gray-300 text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-socialflow-300 focus-visible:ring-offset-1"
                                  >
                                    {next.charAt(0).toUpperCase() + next.slice(1)}
                                  </button>
                                ))}
                              </div>
                            )}

                            {/* Pending transition indicator */}
                            {isPending && (
                              <div className="flex flex-wrap gap-1 mt-2">
                                {allowedNext.map((next) => (
                                  <button
                                    key={next}
                                    disabled
                                    aria-label={`Move to ${next.charAt(0).toUpperCase() + next.slice(1)}`}
                                    className="rounded px-2 py-0.5 text-[10px] font-medium border border-gray-200 text-gray-400 opacity-50 cursor-not-allowed"
                                  >
                                    {next.charAt(0).toUpperCase() + next.slice(1)}
                                  </button>
                                ))}
                              </div>
                            )}

                            {/* Explicit Details & Comments link */}
                            <Link
                              to={`/dashboard/content-items/${item.id}`}
                              className="text-[10px] text-socialflow-600 hover:text-socialflow-700 mt-2 inline-block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-socialflow-300 focus-visible:ring-offset-1 rounded"
                            >
                              Details &amp; Comments →
                            </Link>
                          </div>
                        )
                      })}
                    </div>
                  )}
                </>
              ) : (
                <p className="text-xs text-gray-400">Select a day to see scheduled content</p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
