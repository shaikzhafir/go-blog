# TODOs

- **Slug changes** – If a post’s slug is changed in Notion, old URLs (e.g. `/notion/posts/old-slug`) will 404 because we only resolve the current slug. Leaving as is for now; could add later: redirects from old slugs, or fallback resolution by block ID when the path looks like a UUID.
