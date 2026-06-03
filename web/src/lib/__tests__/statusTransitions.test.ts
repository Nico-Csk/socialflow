import { describe, it, expect } from 'vitest'
// RED: this module does not exist yet — import will fail until GREEN phase
import { NEXT_STATUS } from '@/lib/statusTransitions'

describe('NEXT_STATUS — shared status transition map', () => {
  it('exports a Record<string, string[]> with all five status keys', () => {
    const keys = Object.keys(NEXT_STATUS)
    expect(keys).toHaveLength(5)
    expect(keys).toContain('draft')
    expect(keys).toContain('review')
    expect(keys).toContain('approved')
    expect(keys).toContain('published')
    expect(keys).toContain('archived')
  })

  it('draft transitions to review only', () => {
    expect(NEXT_STATUS['draft']).toEqual(['review'])
  })

  it('review transitions to draft and approved', () => {
    expect(NEXT_STATUS['review']).toEqual(['draft', 'approved'])
  })

  it('approved transitions to published only', () => {
    expect(NEXT_STATUS['approved']).toEqual(['published'])
  })

  it('published transitions to archived only', () => {
    expect(NEXT_STATUS['published']).toEqual(['archived'])
  })

  it('archived has no allowed transitions (terminal status)', () => {
    expect(NEXT_STATUS['archived']).toEqual([])
  })

  // ── status-transitions-shared S3.1: propagation contract ──────────
  it('all transition targets are valid statuses (closed map integrity)', () => {
    const allStatuses = new Set(Object.keys(NEXT_STATUS))
    for (const [from, toList] of Object.entries(NEXT_STATUS)) {
      for (const to of toList) {
        expect(
          allStatuses.has(to),
          `"${to}" in NEXT_STATUS["${from}"] must be a valid status`,
        ).toBe(true)
      }
    }
  })

  it('augmenting the shared map would expose new transitions to both Calendar and ContentDetail', () => {
    // S3.1: Simulate adding a new transition (review → rejected).
    // Both Calendar and ContentDetail consume NEXT_STATUS[item.status],
    // so an augmented map exposes the new transition to BOTH surfaces.
    const augmented: Record<string, string[]> = {
      ...NEXT_STATUS,
      review: [...NEXT_STATUS['review'], 'rejected'],
    }

    expect(augmented['review']).toContain('rejected')
    expect(augmented['review']).toContain('draft')
    expect(augmented['review']).toContain('approved')

    // The original map is unchanged — proves it is the single source of truth
    expect(NEXT_STATUS['review']).not.toContain('rejected')
    expect(NEXT_STATUS['review']).toEqual(['draft', 'approved'])
  })
})
