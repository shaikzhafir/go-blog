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

# credits
- running data provided by [Strava](https://www.strava.com/)
- manga data provided by [MangaDex](https://mangadex.org/)