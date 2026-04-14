## Context

`htmx-blog` is a Go server that renders a personal blog. Notion is the primary CMS: when a post references an image, `services/notion/notion.go::StoreNotionImage` downloads the image from Notion's expiring S3 URL, writes it to `./images/<blockID>.png`, and rewrites the block URL to either `https://cloud.shaikzhafir.com/images/<id>.png` (prod, served by Caddy) or `/images/<id>.png` (dev, served by `http.FileServer` in `cmd/server/main.go`).

Current problems observed:
- Every image is written with a `.png` extension regardless of source format — so JPEGs from Notion get a misleading extension and no compression gains.
- The HTML `<img>` tags in `templates/notion/blocks/image.html`, `templates/pages/reading-now.html`, and `templates/pages/manga.html` have no `loading`, `decoding`, `width`, or `height` attributes. This blocks the browser from lazy-loading and causes layout shift.
- The dev FileServer ships no `Cache-Control` headers. Production is served by Caddy on a different host, so caching policy there is opaque to the code.
- `images/` is already 67MB for 79 files. That will grow.

Stakeholders: just the author (solo blog). No SLA, but reader experience and mobile bandwidth matter.

## Goals / Non-Goals

**Goals:**
- Cut per-image byte count by ≥50% on typical screenshots via WebP conversion (targeting the 60–80% range Google reports for PNG→lossy-WebP; conservatively bank 50%+). Applied to `./images/`, that's roughly 67MB → 15–25MB.
- Eliminate image-driven CLS by declaring intrinsic dimensions on every `<img>`.
- Defer off-screen image loads via `loading="lazy"` while keeping the first image eager for LCP.
- Teach the cache layer (dev FileServer + prod Caddy) to send `Cache-Control: public, max-age=31536000, immutable` for `/images/<id>.<ext>` — safe because IDs are content-addressed and never reused.
- Provide a backfill path for the 79 existing PNGs.

**Non-Goals:**
- AVIF encoding (WebP is enough; AVIF encoding in Go is heavy and slow).
- Responsive `srcset`/`sizes` with multiple widths — the blog layout uses a single rendered width per context; multi-width variants add complexity the traffic doesn't justify yet.
- Self-hosted image CDN or on-the-fly resizing service.
- Migrating reading-now (Goodreads covers) or manga (proxy) image sources to WebP — those are remote URLs we don't control. For those pages, only the template-level attributes (`loading`, `decoding`, `width`, `height`) change.
- Changing the production Caddy config inside this codebase; we document the header and leave the edit to the deploy host.

## Decisions

### 1. WebP encoder: shell out to `cwebp` via `os/exec`

**Chosen:** `os/exec` invoking the `cwebp` binary (libwebp, Google). Quality fixed at `-q 85`. Go code stays pure-stdlib; no Go-level dependency on a WebP library.

**Invocation shape:**
```go
cmd := exec.Command("cwebp", "-q", "85", "-quiet", "-o", outPath, inPath)
```
The encoder reads the original bytes already written to disk (step 2 of the storage path) and writes `./images/<id>.webp`. Image dimensions are still obtained in-process via `image.DecodeConfig` from stdlib (`image/png`, `image/jpeg`, `image/gif`) — no need to parse `cwebp` output for that.

**Alternatives considered:**
- `github.com/chai2010/webp` (cgo wrapper around libwebp) — rejected because cgo complicates cross-compilation and adds a Go-level dependency for something a one-line binary install handles.
- `github.com/kolesa-team/go-webp` — same cgo tradeoff.
- Pure-Go encoder (`github.com/HugoSmits86/nativewebp`) — rejected because output quality/size is measurably worse than libwebp, and the point of this change is byte reduction.
- `golang.org/x/image/webp` — decode-only, not usable here.

**Trade-off:** requires the `cwebp` binary on the host. The deploy VPS has it installed (`apt install webp`); a dev without it gets a clean error at encode time and the fallback-only path kicks in (see decision 3).

### 2. Detect source format from response, not filename

Notion's S3 URLs don't always include an extension. Use `http.DetectContentType` on the first 512 bytes of the download. Reliably WebP-encoded inputs: PNG, JPEG. GIF and anything else `cwebp` can't open (brew/Debian builds of `cwebp` omit libgif by default) fall into the encode-failure path: the original bytes are kept and the URL points at them with a plain `<img>` (no `<picture>`).

### 3. Always produce WebP + keep original as fallback

Write two files:
- `./images/<id>.webp` — the optimized output, used as the primary `src`.
- `./images/<id>.<origExt>` — the raw download, referenced as `<source>` fallback inside `<picture>` for UAs without WebP support (very small tail now, but zero marginal cost).

If WebP encoding fails, fall back to only the original and emit a plain `<img>`.

### 4. Template change: `<picture>` with WebP source

`templates/notion/blocks/image.html` becomes:
```html
<figure class="my-10 flex justify-center">
  <picture>
    <source srcset="{{.WebpURL}}" type="image/webp" />
    <img src="{{.FallbackURL}}" alt="post image" loading="lazy" decoding="async"
         width="{{.Width}}" height="{{.Height}}" class="..." />
  </picture>
</figure>
```

This requires threading `WebpURL`, `FallbackURL`, `Width`, `Height` through the notion block renderer. The renderer currently hands raw block JSON to the template — extend the image-block render path to produce a struct with these fields.

Width/height come from decoding the image once during storage (we already have the bytes in memory) and caching the dimensions alongside the file. Store as a sidecar `.meta.json` (or extend an existing metadata store if one exists in `services/notion`). Simpler: encode the dimensions into the filename, e.g. `<id>_<w>x<h>.webp`, so the template can derive them without a second read. **Chosen:** sidecar JSON, since the URL is referenced from cached block JSON that would be painful to rewrite; a small `<id>.meta.json` read at render time is cheap.

### 5. Lazy-load everywhere except the first visible image

- Notion block image: `loading="lazy"` unconditionally — the first image in a post is usually below a heading and title, so above-the-fold is the text, not the image.
- `templates/pages/reading-now.html`: add `loading="lazy"` to list items beyond the first via a `{{if eq $index 0}}eager{{else}}lazy{{end}}` pattern, OR just set all to `lazy` if the first card is below the fold on mobile. **Chosen:** all `lazy` — the hero here is text, not the cover image.
- `templates/pages/manga.html`: all `lazy`.

### 6. Cache headers

- **Dev:** wrap the `/images/` FileServer handler with a middleware that sets `Cache-Control: public, max-age=31536000, immutable` before delegating. Inline in `cmd/server/main.go`.
- **Prod:** content IDs are stable (Notion block IDs don't change), so `immutable` is correct. Document the required Caddy directive in the proposal/design; the change does not modify Caddy config.

### 7. Backfill

One-off admin handler behind `internalMux` (already used for cron): `POST /cron/backfill-images` iterates `./images/*.png`, for each file without a matching `<id>.webp` encode and write it, plus write the sidecar metadata. Idempotent; safe to re-run.

## Risks / Trade-offs

- [Host missing `cwebp` binary] → Encode step returns an error and the fallback-only path takes over (plain `<img>` pointing at the original PNG/JPEG). Mitigation: log prominently on first failure; document `apt install webp` in deploy docs.
- [WebP encoding CPU cost on first fetch] → Notion image sync happens on cache refresh, not on request path, so user latency is unaffected. Fork/exec per image is ~50–100ms, dwarfed by the S3 download itself. Mitigation: only encode once; check for existing `.webp` before re-encoding.
- [Browsers without WebP support] → <2% global, but the `<picture>` fallback covers them at zero cost.
- [Sidecar metadata goes out of sync] → If `.webp` exists but `.meta.json` missing, render path falls back to plain `<img src=fallback>` without dimensions (accepts CLS for that one image). Not fatal.
- [Existing block JSON in cache still has `.png` URL] → On next cache refresh, block storage rewrites URL to `.webp`. Between now and then, readers see the old `.png`. Acceptable; optional cache flush can accelerate.
- [Backfill handler left exposed] → `internalMux` is assumed bound to localhost only; verify before shipping.

## Migration Plan

1. Add dependency and encoder wrapper; unit-test encode round-trip.
2. Update `StoreNotionImage` (and the sibling helper near `notion.go:482`) to write both the WebP and the fallback, plus sidecar metadata. Rewrite URL to the `.webp`.
3. Update `templates/notion/blocks/image.html` to a `<picture>` element; update block renderer to pass WebP + fallback + dimensions.
4. Add `loading="lazy"` / `decoding="async"` / `width` / `height` to all remaining `<img>` tags (reading-now, manga).
5. Add cache-header middleware to dev FileServer; document Caddy header in deploy docs.
6. Ship backfill handler; run once against `./images/`.
7. Trigger a cache refresh so block JSON picks up the new URLs.
8. Verify: Lighthouse LCP/CLS on a post with images before and after; bytes shipped per image drops ≥50%.

**Rollback:** revert the template change and URL rewrite. The original `.png` files stay on disk, so old URLs keep working even if encoder code is reverted.

## Open Questions

- Does `services/notion` already have a metadata store we can extend, or is a new sidecar `.meta.json` the cleanest option? (Leaning sidecar; confirm during implementation.)
- Is the production Caddy config in this repo or elsewhere? If elsewhere, this change is docs-only on the prod caching front.
