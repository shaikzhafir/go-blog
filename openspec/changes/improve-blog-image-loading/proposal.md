## Why

Blog posts, reading-now, and manga pages render images with no lazy loading, no size hints, and no modern format negotiation. Notion-sourced images are stored as raw `.png` regardless of source format, so a single post can ship several hundred KB to multiple MB of image bytes on first paint. The current `images/` directory is already ~67MB for 79 files (avg ~850KB). This hurts LCP, causes layout shift, and wastes bandwidth for the reader and for the cloud host serving `cloud.shaikzhafir.com`.

### Expected size savings from WebP

Published benchmarks (Google, 2020; corroborated by web.dev, Cloudflare, and Smashing Magazine studies):

| Source format | Typical WebP saving | Notes |
|---|---|---|
| PNG (screenshots, UI, flat color) → lossy WebP @ q80 | **60–80% smaller** | Biggest win; PNG is uncompressed-ish for photographic/gradient content |
| PNG → lossless WebP | ~26% smaller | Safe option, still a real win |
| JPEG (photos) → lossy WebP @ q80 | 25–34% smaller at equivalent SSIM | Google's original WebP study |
| GIF (static) → WebP | 64% smaller lossy, 19% smaller lossless | |

Most Notion images are screenshots pasted from macOS/Linux — PNG with flat regions and text. Those fall in the 60–80% bucket. Applied to the current directory: **67MB → ~15–25MB** (estimated 60–75% reduction), and a typical 850KB image drops to ~170–340KB. Per-post bytes on the wire should drop by a similar fraction, since Notion images dominate post payload today.

Photographic content (rare on this blog) sees a smaller but still worthwhile 25–35% cut.

## What Changes

- Add `loading="lazy"` and `decoding="async"` attributes to non-critical `<img>` tags in Notion blocks, reading-now, and manga templates; keep the first/above-the-fold image eager.
- Add intrinsic `width` and `height` (or aspect-ratio CSS) to `<img>` tags so the browser reserves space and avoids CLS.
- Convert Notion images to WebP during `StoreNotionImage` download, with the original PNG kept only as a fallback if conversion fails. New URLs use `.webp`.
- Emit a `<picture>` element in `templates/notion/blocks/image.html` with a WebP source plus PNG/JPEG fallback for older clients.
- Add long-lived `Cache-Control` headers on the dev `/images/` FileServer, and document the equivalent header for the production Caddy/Nginx host.
- Backfill: one-off script/handler to re-encode existing images in `./images/` to WebP.

## Capabilities

### New Capabilities
- `blog-image-delivery`: how the blog stores, transforms, and serves images used in Notion posts and other blog pages — covers format, caching, and HTML emission contracts.

### Modified Capabilities
<!-- None: no existing spec files under openspec/specs/ to modify. -->

## Impact

- Affected code:
  - `services/notion/notion.go` (`StoreNotionImage`, the earlier inline image-download helper near line 482)
  - `templates/notion/blocks/image.html`
  - `templates/pages/reading-now.html`
  - `templates/pages/manga.html`
  - `cmd/server/main.go` (dev `/images/` FileServer; optional cache-header middleware)
- Dependencies: no new Go-level dependencies. Encoding shells out to the `cwebp` binary via `os/exec`; image dimensions read via stdlib `image/png`, `image/jpeg`, `image/gif`. Host requirement: `cwebp` installed (`apt install webp` — already present on the prod VPS).
- Ops: existing `./images/` contents must be migrated; production Caddy/Nginx serving `cloud.shaikzhafir.com/images/` needs a `Cache-Control: public, max-age=31536000, immutable` header. No DB changes.
- No breaking changes for readers; old `.png` URLs stay valid during and after migration.
