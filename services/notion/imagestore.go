package notion

import (
	"encoding/json"
	"fmt"
	log "htmx-blog/logging"
	"htmx-blog/services/notion/imageenc"
	"net/http"
	"os"
	"path/filepath"
)

// ImageMeta is the sidecar written alongside every stored image.
// Lives at ./images/<id>.meta.json.
type ImageMeta struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	FallbackExt string `json:"fallbackExt"` // "png" / "jpg" / "gif" / etc, no dot
	HasWebP     bool   `json:"hasWebP"`
}

// imagesDir returns an absolute path to ./images, creating it if needed.
func imagesDir() (string, error) {
	abs, err := filepath.Abs("./images")
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %v", err)
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		if err := os.Mkdir(abs, os.ModePerm); err != nil {
			return "", fmt.Errorf("error creating images dir: %v", err)
		}
	}
	return abs, nil
}

// extFromContentType maps a detected MIME type to a file extension without the leading dot.
// Returns the zero value "" for unrecognized types so callers can fall back.
func extFromContentType(ct string) string {
	switch ct {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return ""
}

// storeImageBytes writes the original bytes under <id>.<ext>, encodes a WebP
// sibling at <id>.webp when possible, and writes an <id>.meta.json sidecar.
// Returns the URL that templates should link to (WebP when available, else
// the original) and the written meta for callers that need dimensions.
//
// Encode failures are logged and absorbed; the fallback path is always viable.
func storeImageBytes(id string, body []byte) (string, ImageMeta, error) {
	dir, err := imagesDir()
	if err != nil {
		return "", ImageMeta{}, err
	}

	sniff := body
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	ct := http.DetectContentType(sniff)
	ext := extFromContentType(ct)
	if ext == "" {
		// Unrecognized image — keep the old .png convention so nothing breaks,
		// but flag it loudly. Rendering path will still show the plain img.
		log.Error("unrecognized image content-type %q for id %s; storing as .png", ct, id)
		ext = "png"
	}

	origPath := filepath.Join(dir, id+"."+ext)
	if err := os.WriteFile(origPath, body, 0o755); err != nil {
		return "", ImageMeta{}, fmt.Errorf("error writing image to file: %v", err)
	}

	meta := ImageMeta{FallbackExt: ext}

	// Dimensions: read from whatever we just wrote. Best-effort.
	if w, h, derr := imageenc.ReadDimensions(origPath); derr == nil {
		meta.Width, meta.Height = w, h
	} else {
		log.Error("could not read image dimensions for %s: %v", id, derr)
	}

	// WebP encode: best-effort. If already webp, skip (original IS the webp).
	if ext == "webp" {
		meta.HasWebP = true
	} else {
		webpPath := filepath.Join(dir, id+".webp")
		if err := imageenc.EncodeWebP(origPath, webpPath); err != nil {
			log.Error("webp encode failed for %s: %v", id, err)
		} else {
			meta.HasWebP = true
		}
	}

	// Write sidecar.
	metaPath := filepath.Join(dir, id+".meta.json")
	if metaBytes, merr := json.Marshal(meta); merr == nil {
		if werr := os.WriteFile(metaPath, metaBytes, 0o644); werr != nil {
			log.Error("error writing image meta sidecar for %s: %v", id, werr)
		}
	}

	// Decide the primary URL: webp if we have one, else the fallback.
	primaryExt := ext
	if meta.HasWebP && ext != "webp" {
		primaryExt = "webp"
	}
	return imageURLFor(id, primaryExt), meta, nil
}

// imageURLFor returns the absolute URL in prod, root-relative in dev.
func imageURLFor(id, ext string) string {
	if os.Getenv("DEV") == "true" {
		return "/images/" + id + "." + ext
	}
	return "https://cloud.shaikzhafir.com/images/" + id + "." + ext
}

// readImageMeta loads the sidecar for id. Returns a zero ImageMeta and nil
// error if the sidecar is missing, so callers can treat "no sidecar" the same
// as "no WebP, no dimensions" without branching on os.IsNotExist.
func readImageMeta(id string) (ImageMeta, error) {
	dir, err := imagesDir()
	if err != nil {
		return ImageMeta{}, err
	}
	path := filepath.Join(dir, id+".meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ImageMeta{}, nil
		}
		return ImageMeta{}, err
	}
	var meta ImageMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return ImageMeta{}, err
	}
	return meta, nil
}
