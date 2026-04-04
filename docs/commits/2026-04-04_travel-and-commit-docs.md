# Travel nav + per-commit documentation

**Commit:** See `git log -1 --oneline` on the branch that contains this file (this note ships in the same commit as the feature).

## Context

Add a travel section that uses the same Notion data source as other posts, filtered by the `travel` tag. Also introduce a convention so meaningful commits can be tracked with a short plan-to-shipped note in-repo.

## Plan

1. Add `docs/commits/README.md` describing file naming, suggested sections, optional `_wip_` files, and optional retroactive backfill.
2. Add a primary nav link to `/notion/travel` so lists and posts use filter `travel` (existing handlers and templates already pass `type` when `PostType` is set).

## Implementation

- Added `docs/commits/` README for how to write per-commit notes.
- Added a **travel** nav item in the layout after **coding**, before **strava**, matching other Notion links (`data-nav`, same classes). No new routes: `GET /notion/{filter}` already queries Notion `tags` multi-select `contains` for the path segment.

## Files touched

- `docs/commits/README.md`
- `docs/commits/2026-04-04_travel-and-commit-docs.md`
- `templates/layout/main.html`

## Verify

1. Open `docs/commits/README.md` and confirm the workflow is what you want going forward.
2. In Notion, tag pages with `travel` (exact spelling; case-sensitive).
3. Visit `/notion/travel`; open a post and confirm HTMX loads content with `?type=travel`.
4. Confirm the **travel** header link shows active state on `/notion/travel` and nested post paths.
