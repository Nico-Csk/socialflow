/**
 * Canonical status transition map shared by Calendar and ContentDetail.
 * Each key is a current status; the value is the array of allowable next statuses.
 */
export const NEXT_STATUS: Record<string, string[]> = {
  draft: ['review'],
  review: ['draft', 'approved'],
  approved: ['published'],
  published: ['archived'],
  archived: [],
}
