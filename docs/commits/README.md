# Commit notes

One file per commit that matters, so intent and what shipped stay easy to find.

## Naming

After you merge or push, file name:

`YYYY-MM-DD_<short-sha>_<topic-slug>.md`

Use a 7-character short SHA (`git rev-parse --short HEAD`). Replace `<topic-slug>` with a few lowercase words (e.g. `travel-nav`, `commit-docs-convention`).

If the note is in the **same commit** as the code, use `YYYY-MM-DD_<topic-slug>.md` instead (no SHA in the filename) so you do not need an extra amend; mention `git log -1 --oneline` in the note if you want the hash.

While work is in flight, you may use `docs/commits/_wip_<topic>.md`, then rename when the commit exists.

## What to write

- **Context** — why the change.
- **Plan** — what you intended before coding (can be short).
- **Implementation** — what actually landed.
- **Files touched** — paths relative to repo root.
- **Verify** — commands or URLs to confirm behavior.

Update the file after the commit so **Implementation** and **Files touched** match the real diff.

## Retroactive history

Backfilling old commits is optional. New work can start documenting from the first entry in this directory.
