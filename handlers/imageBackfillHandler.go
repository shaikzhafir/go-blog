package handlers

import (
	"fmt"
	log "htmx-blog/logging"
	"htmx-blog/services/notion"
	"net/http"
)

// ImageBackfillHandler returns a handler that scans ./images/ and ensures
// every non-WebP file has a WebP sibling and a metadata sidecar. Idempotent.
// Intended to live on the internal mux (localhost-only).
func ImageBackfillHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := notion.BackfillImages()
		if err != nil {
			log.Error("image backfill failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "backfill complete: scanned=%d encoded=%d skipped=%d failed=%d\n",
			report.Scanned, report.Encoded, report.Skipped, report.Failed)
	}
}
