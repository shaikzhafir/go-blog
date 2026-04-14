# go-htmx-blog

## Run the site

```bash
make deps          # go mod tidy
make run           # build and run server
```

**Development (hot reload):** run in two terminals:

```bash
make dev           # Go server with entr (reloads on .go / .html changes)
make tailwind      # Tailwind CSS watch (rebuilds on input.css / template changes)
```

See `make help` for all targets.

## Tailwind CSS

1. Download the [Tailwind standalone CLI](https://tailwindcss.com/blog/standalone-cli) and place `tailwindcss` in the repo root (or set `TAILWIND_CLI` in the Makefile).
2. **Dev:** `make tailwind` (watch mode).
3. **Production:** `make css` (minified build).

## Images

Notion-sourced images are downloaded once, re-encoded to WebP (quality 85) via
the `cwebp` binary, and served from `./images/`. Host requirement:

```bash
apt install webp   # or: brew install webp
```

If `cwebp` is missing the server still runs — new images just skip WebP encoding
and are served in their original format. Log will warn on startup.

### Production caching

The reverse proxy serving `cloud.shaikzhafir.com/images/` **must** set a
long-lived immutable cache header, because image filenames are content-addressed
(Notion block ID) and never reused:

```
Cache-Control: public, max-age=31536000, immutable
```

Caddy example:

```
cloud.shaikzhafir.com {
  @images path /images/*
  header @images Cache-Control "public, max-age=31536000, immutable"
  root * /srv/shaikzhafir
  file_server
}
```

The dev Go server already sets this header on its `/images/` file server.

### Backfilling existing images to WebP

After deploying, run once against the internal cron endpoint to encode any
pre-existing PNG/JPEGs:

```bash
curl -X POST http://127.0.0.1:8081/cron/backfill-images
```

Idempotent; re-running after new images land is safe.

# credits
- running data provided by [Strava](https://www.strava.com/)
- manga data provided by [MangaDex](https://mangadex.org/)