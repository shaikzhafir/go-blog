package notion

import (
	"encoding/json"
	log "htmx-blog/logging"
	"htmx-blog/services/notion/imageenc"
	"os"
	"path/filepath"
	"strings"
)

// BackfillReport summarises a backfill run.
type BackfillReport struct {
	Scanned int
	Encoded int
	Skipped int
	Failed  int
}

// BackfillImages walks ./images/ and, for every non-WebP, non-meta file that
// doesn't already have a <id>.webp sibling, encodes the WebP and writes the
// sidecar. Files that already have a WebP sibling are left alone.
func BackfillImages() (BackfillReport, error) {
	var report BackfillReport
	dir, err := imagesDir()
	if err != nil {
		return report, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return report, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".webp" || ext == ".json" {
			continue
		}
		report.Scanned++

		id := strings.TrimSuffix(name, filepath.Ext(name))
		origPath := filepath.Join(dir, name)
		webpPath := filepath.Join(dir, id+".webp")
		metaPath := filepath.Join(dir, id+".meta.json")

		webpExists := fileExists(webpPath)
		metaExists := fileExists(metaPath)
		if webpExists && metaExists {
			report.Skipped++
			continue
		}

		// Derive fallback extension (without leading dot) from filename.
		fallbackExt := strings.TrimPrefix(ext, ".")

		meta := ImageMeta{FallbackExt: fallbackExt}
		if w, h, derr := imageenc.ReadDimensions(origPath); derr == nil {
			meta.Width, meta.Height = w, h
		} else {
			log.Error("backfill: could not read dimensions for %s: %v", name, derr)
		}

		if !webpExists {
			if err := imageenc.EncodeWebP(origPath, webpPath); err != nil {
				log.Error("backfill: webp encode failed for %s: %v", name, err)
				report.Failed++
			} else {
				meta.HasWebP = true
				report.Encoded++
			}
		} else {
			meta.HasWebP = true
		}

		if metaBytes, merr := json.Marshal(meta); merr == nil {
			if werr := os.WriteFile(metaPath, metaBytes, 0o644); werr != nil {
				log.Error("backfill: error writing sidecar for %s: %v", name, werr)
			}
		}
	}

	return report, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
