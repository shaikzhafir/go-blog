## 1. Encoder wrapper (stdlib only + cwebp binary)

- [x] 1.1 Confirm `cwebp` is on `PATH` at startup (log a warning if `exec.LookPath("cwebp")` fails); no Go module changes needed
- [x] 1.2 Create `services/notion/imageenc/` (or a sibling package) exposing `EncodeWebP(srcPath, dstPath string) error` that shells out `cwebp -q 85 -quiet -o <dst> <src>` and returns a wrapped error on non-zero exit, plus `ReadDimensions(srcPath string) (w, h int, err error)` using stdlib `image.DecodeConfig`
- [x] 1.3 Unit-test the wrapper against fixture PNG and JPEG inputs (assert `.webp` output exists, is non-empty, has the WebP magic bytes `RIFF....WEBP`, and that dimensions match); add a test that stubs `cwebp` missing and verifies the error path. (GIF dropped — `cwebp` on brew/Debian doesn't link libgif; GIFs fall into the encode-failure path instead.)

## 2. Storage path: write WebP + fallback + sidecar

- [x] 2.1 Refactor `services/notion/notion.go::StoreNotionImage` to:
  - detect content type from the first 512 bytes (`http.DetectContentType`)
  - choose the correct fallback extension (`.png`, `.jpg`, `.gif`, `.webp`)
  - write `./images/<blockID>.<ext>` (original bytes) and `./images/<blockID>.webp` (encoded)
  - write `./images/<blockID>.meta.json` with `{width, height, fallbackExt}`
  - rewrite `imageBlock.Image.File.URL` to the `.webp` URL (dev vs prod host), matching existing conditional
- [x] 2.2 Apply the same refactor to the earlier inline image helper near `services/notion/notion.go:482` (now `convertAndStoreImage`)
- [x] 2.3 Handle the WebP-encode-failure path: log, keep original, rewrite URL to the fallback file, do not bubble the error up
- [x] 2.4 Add/extend unit tests in `services/notion/` covering: PNG source, JPEG source, encoder error path, and sidecar contents

## 3. Block renderer: thread WebP + fallback + dimensions to template

- [x] 3.1 Inspect `services/notion` block renderer (`notion.NewBlockRenderer`) and identify where image blocks resolve to template data — it's `converter.RenderImage` in `notion.go`
- [x] 3.2 Extend the image-block render struct with `WebpURL`, `FallbackURL`, `Width`, `Height`, `HasWebP` fields; populate by reading `<id>.meta.json` and constructing URLs via `imageURLFor(id, ext)` so stale cached block URLs don't matter
- [x] 3.3 Gracefully degrade when sidecar/WebP are missing: `HasWebP=false`, dimensions zeroed, `FallbackURL` falls back to the URL stored in the block

## 4. Template updates

- [x] 4.1 Rewrite `templates/notion/blocks/image.html` to emit a `<picture>` element when `HasWebP`, otherwise a plain `<img>`; always include `loading="lazy"`, `decoding="async"`, and numeric `width`/`height` when non-zero
- [x] 4.2 Update `templates/pages/reading-now.html` `<img>` with `loading="lazy"`, `decoding="async"`, and `width="112" height="160"` (matches `w-28 h-40`)
- [x] 4.3 Update `templates/pages/manga.html` `<img>` with `loading="lazy"`, `decoding="async"`, and aspect-ratio style

## 5. Cache headers

- [x] 5.1 In `cmd/server/main.go`, wrap the `/images/` FileServer in a middleware that sets `Cache-Control: public, max-age=31536000, immutable` before serving
- [x] 5.2 Add or update a deploy doc describing the required Caddy/Nginx header for `cloud.shaikzhafir.com/images/` (added to `README.md`)

## 6. Backfill

- [x] 6.1 Add internal handler `POST /cron/backfill-images` on `internalMux` in `cmd/server/main.go`
- [x] 6.2 Implement the handler: walk `./images/`, for each non-WebP file without a matching `.webp` sibling, encode and write WebP + sidecar; skip otherwise
- [x] 6.3 Run the handler once against the existing PNGs — 79 scanned, 79 encoded, 0 skipped, 0 failed. PNG total 67MB → WebP total 11MB (84% reduction)

## 7. Verification

- [ ] 7.1 Manually load a post with images in dev; confirm DevTools Network panel shows `image/webp` responses with `Cache-Control` header set
- [ ] 7.2 Lighthouse before/after on a representative post: record LCP and CLS, confirm improvement (LCP down, CLS → ~0)
- [ ] 7.3 Smoke-test reading-now and manga pages: images load lazily as the viewport scrolls
- [x] 7.4 ~~Trigger a Notion cache refresh so block JSON URLs update to `.webp`~~ — **not required**. The renderer constructs URLs from `block.ID` + sidecar (not from the cached URL), so existing cached posts render WebP immediately after backfill.
