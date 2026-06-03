// ── Filter Constants ───────────────────────────────────────────────
export const STATUS_OPTIONS = ['all', 'draft', 'review', 'approved', 'published', 'archived'] as const
export const PLATFORM_OPTIONS = ['all', 'instagram', 'facebook', 'twitter', 'linkedin', 'tiktok', 'youtube', 'other'] as const

// ── Query Builder ───────────────────────────────────────────────────
export function buildCalendarQuery(params: {
  month: string
  status?: string
  platform?: string
}): string {
  const parts: string[] = [`month=${params.month}`]
  if (params.status && params.status !== 'all') {
    parts.push(`status=${params.status}`)
  }
  if (params.platform && params.platform !== 'all') {
    parts.push(`platform=${params.platform}`)
  }
  return `/calendar?${parts.join('&')}`
}
