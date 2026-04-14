## ADDED Requirements

### Requirement: Notion images are stored in WebP with a fallback copy

The system SHALL, when ingesting an image block from Notion, download the source image, encode a WebP version at quality ≤ 85, and persist both the WebP output and the original-format copy to the images directory under content-addressed filenames keyed by the Notion block ID.

#### Scenario: PNG source converts to WebP plus PNG fallback
- **WHEN** `StoreNotionImage` runs against a Notion block whose source URL returns PNG bytes
- **THEN** the filesystem contains `./images/<blockID>.webp` (the encoded WebP) and `./images/<blockID>.png` (the original bytes)
- **AND** the block's rewritten URL points at the `.webp` file

#### Scenario: JPEG source keeps correct fallback extension
- **WHEN** `StoreNotionImage` runs against a source that returns JPEG bytes (detected via `http.DetectContentType`)
- **THEN** the fallback file is written as `./images/<blockID>.jpg`, not `.png`
- **AND** the WebP file is still produced at `./images/<blockID>.webp`

#### Scenario: WebP encoding failure falls back to original only
- **WHEN** WebP encoding returns an error for a given image
- **THEN** the system persists only the original-format file
- **AND** the block's rewritten URL points at the original file
- **AND** the storage step does not return an error (the image is still renderable)

### Requirement: Stored images have a sidecar with intrinsic dimensions

The system SHALL record the pixel `width` and `height` of each stored image in a sidecar metadata file `./images/<blockID>.meta.json`, readable by templates at render time, so HTML can declare intrinsic dimensions and avoid layout shift.

#### Scenario: Sidecar is written alongside the image
- **WHEN** an image is successfully stored
- **THEN** `./images/<blockID>.meta.json` exists and contains at minimum `{"width": <int>, "height": <int>}`

#### Scenario: Missing sidecar does not break rendering
- **WHEN** a template renders an image whose `.meta.json` is missing or unreadable
- **THEN** the template emits an `<img>` without `width`/`height` attributes rather than failing

### Requirement: Notion image blocks render as a `<picture>` with WebP source and fallback

The system SHALL render Notion image blocks using a `<picture>` element that offers the WebP file as a typed `<source>` and the original-format file as the `<img>` fallback, with `loading="lazy"`, `decoding="async"`, and `width`/`height` attributes when dimensions are known.

#### Scenario: Both formats available
- **WHEN** a post containing an image block is rendered and both `.webp` and fallback files exist with a readable sidecar
- **THEN** the output contains a `<picture>` element with a `<source type="image/webp" srcset="...">` pointing at the `.webp` URL and an `<img src="...">` pointing at the fallback URL
- **AND** the `<img>` has `loading="lazy"`, `decoding="async"`, and numeric `width` and `height` attributes

#### Scenario: WebP missing
- **WHEN** only the fallback file exists (e.g., encoding failed during ingest)
- **THEN** the output contains a plain `<img>` with `src` pointing at the fallback URL, `loading="lazy"`, and `decoding="async"`

### Requirement: Non-Notion image templates declare loading and sizing hints

The system SHALL apply `loading="lazy"`, `decoding="async"`, and explicit `width` and `height` (or an equivalent `aspect-ratio` style that reserves space) to every `<img>` tag rendered by the reading-now and manga templates.

#### Scenario: Reading-now cover images
- **WHEN** the reading-now page renders a book card
- **THEN** each cover `<img>` has `loading="lazy"`, `decoding="async"`, and width/height (or aspect-ratio) attributes

#### Scenario: Manga cover images
- **WHEN** the manga page renders a manga card
- **THEN** each cover `<img>` has `loading="lazy"`, `decoding="async"`, and width/height (or aspect-ratio) attributes

### Requirement: `/images/` responses carry long-lived cache headers

The system SHALL serve files under `/images/` with `Cache-Control: public, max-age=31536000, immutable` in the development FileServer, and SHALL document the same header requirement for the production reverse proxy serving `cloud.shaikzhafir.com/images/`.

#### Scenario: Dev FileServer response
- **WHEN** a client requests `/images/<id>.webp` from the Go dev server
- **THEN** the response includes `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: Production caching documented
- **WHEN** a maintainer reads the project's deploy documentation
- **THEN** it states that the reverse proxy for `cloud.shaikzhafir.com/images/` must set `Cache-Control: public, max-age=31536000, immutable`

### Requirement: Backfill converts existing images idempotently

The system SHALL provide an internal, idempotent operation that scans `./images/` for files lacking a matching `.webp` sibling and produces the WebP plus sidecar for each, without re-encoding files that already have a WebP sibling.

#### Scenario: First run encodes missing WebPs
- **WHEN** the backfill operation runs and `./images/abc.png` exists but `./images/abc.webp` does not
- **THEN** after the run, `./images/abc.webp` exists and `./images/abc.meta.json` exists

#### Scenario: Second run is a no-op
- **WHEN** the backfill operation runs a second time with no filesystem changes between runs
- **THEN** no files are re-encoded or rewritten
