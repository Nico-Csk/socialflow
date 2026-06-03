# Exploration: Calendar Filter UX

## Current State

The Calendar page (`web/src/pages/Calendar/Calendar.tsx`) renders a monthly grid with
Prev/Next navigation, a "today" highlight on the current date, and a selected-day
sidebar. However, **it sends zero filter parameters** to the backend — only `?month=YYYY-MM`.

Meanwhile, the backend **fully supports** three optional filters on
`GET /api/calendar?month=YYYY-MM&platform=instagram&status=draft&client_id=uuid`:

- `status` — validated against domain.ContentStatus (5 values)
- `platform` — validated against domain.ContentPlatform (7 values)
- `client_id` — UUID, scoped to workspace

The store layer builds dynamic SQL WHERE clauses for each non-nil filter. No backend
changes are needed.

The existing **ContentList.tsx** component demonstrates a canonical filter pattern:
`useSearchParams` for URL-synced status filter tabs. The Calendar currently uses
plain `useState` for year/month/selectedDate — no URL persistence.

## Affected Areas

- `web/src/pages/Calendar/Calendar.tsx` — main target: add filter controls, URL sync, better UX
- `web/src/pages/ContentItems/ContentList.tsx` — reference pattern for status tabs with useSearchParams
- `web/src/pages/ContentItems/ContentForm.tsx` — platform enum reference (7 platforms)
- `web/src/pages/__tests__/Calendar.test.tsx` — update tests for filter behavior
- `internal/domain/content.go` — source of truth for enums (no changes needed)
- `internal/http/handler_calendar.go` — backend handler (no changes needed)
- `internal/service/content.go` — CalendarParams/ListByMonth (no changes needed)
- `internal/store/content_items.go` — dynamic SQL filters (no changes needed)

## Approaches

### A. Minimal — Filter controls + URL sync
Add status filter tabs (like ContentList) and basic URL sync for month. No platform filter.
- Pros: Fast, follows existing pattern, low risk
- Cons: Doesn't address missing platform filter; selected-day panel still limited
- Effort: Low

### B. Full — Filter controls + panel redesign + URL sync ⭐ **RECOMMENDED**
- `useSearchParams` for month, status, platform (follows ContentList pattern)
- Status filter tabs (reuse ContentList pattern)
- Platform filter (tag-style selector or dropdown)
- "Today" button to jump back to current month
- Selected-day panel always visible with "Select a day" hint
- Better empty/loading states
- Active filter badges/chips showing current state
- Optional: client filter dropdown
- Pros: Covers all UX gaps, follows established patterns, no backend changes
- Cons: Platform filter is a new pattern (not in ContentList)
- Effort: Medium

### C. Premium — Calendar redesign
Everything in B plus skeleton loading, animations, drag-and-drop scheduling, week view toggle, color legend.
- Pros: Polished UX
- Cons: Over-engineering for current MVP stage; drag-and-drop needs backend changes
- Effort: High

## Recommendation

**Approach B** — Full but practical. It follows the existing `useSearchParams` pattern
from ContentList, adds the genuinely missing platform filter, improves the selected-day
panel, and adds "Today" affordance. This is appropriate for a client-facing MVP without
over-engineering.

No backend changes required. Everything lives in the React frontend.

## Risks

- Platform filter is a new pattern (ContentList doesn't have it) — needs careful component design
- URL sync for month means filter changes trigger re-fetch; need to handle stale data or use debounce
- Client filter depends on client list being available; may need a separate API call for dropdown options
- Existing test mocks will need updating when filter interactions change
- The selected-day panel becoming always-visible changes layout on initial render

## Ready for Proposal

Yes. The investigation is complete and the recommended approach (B) is clearly scoped.
The orchestrator can proceed to the propose phase with confidence.
