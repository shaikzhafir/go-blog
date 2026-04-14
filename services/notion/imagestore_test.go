package notion

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// imagestore_test.go tests the refactored storage helper directly against
// in-memory-generated PNG/JPEG bytes, so it doesn't need Notion mocks or
// network. cwebp must be on PATH for the WebP branch to exercise; otherwise
// we verify the fallback path still produces a valid sidecar.

func tinyImageBytes(t *testing.T, enc func(w *bytes.Buffer, img image.Image) error) []byte {
	t.Helper()
	const w, h = 32, 24
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := enc(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// chdirTo relocates the process cwd for the test since storeImageBytes
// writes under "./images". Restores cwd on cleanup.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func TestStoreImageBytes_PNG(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not on PATH; skipping")
	}
	chdirTo(t, t.TempDir())
	// DEV=true so URL is relative, easier to assert.
	t.Setenv("DEV", "true")

	pngBytes := tinyImageBytes(t, func(w *bytes.Buffer, img image.Image) error { return png.Encode(w, img) })
	url, meta, err := storeImageBytes("test-png-id", pngBytes)
	if err != nil {
		t.Fatalf("storeImageBytes: %v", err)
	}
	if url != "/images/test-png-id.webp" {
		t.Fatalf("url: want /images/test-png-id.webp, got %s", url)
	}
	if meta.FallbackExt != "png" {
		t.Fatalf("fallbackExt: want png, got %s", meta.FallbackExt)
	}
	if !meta.HasWebP {
		t.Fatal("expected HasWebP=true")
	}
	if meta.Width != 32 || meta.Height != 24 {
		t.Fatalf("dims: want 32x24, got %dx%d", meta.Width, meta.Height)
	}

	// Both files should exist on disk.
	for _, p := range []string{"images/test-png-id.png", "images/test-png-id.webp", "images/test-png-id.meta.json"} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s: %v", p, err)
		}
	}

	// Sidecar should round-trip.
	data, err := os.ReadFile("images/test-png-id.meta.json")
	if err != nil {
		t.Fatal(err)
	}
	var got ImageMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != meta {
		t.Fatalf("sidecar mismatch: %+v vs %+v", got, meta)
	}
}

func TestStoreImageBytes_JPEG(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not on PATH; skipping")
	}
	chdirTo(t, t.TempDir())
	t.Setenv("DEV", "true")

	jpegBytes := tinyImageBytes(t, func(w *bytes.Buffer, img image.Image) error {
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
	})
	url, meta, err := storeImageBytes("test-jpg-id", jpegBytes)
	if err != nil {
		t.Fatalf("storeImageBytes: %v", err)
	}
	if url != "/images/test-jpg-id.webp" {
		t.Fatalf("url: got %s", url)
	}
	if meta.FallbackExt != "jpg" {
		t.Fatalf("fallbackExt: want jpg, got %s", meta.FallbackExt)
	}
	for _, p := range []string{"images/test-jpg-id.jpg", "images/test-jpg-id.webp"} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s: %v", p, err)
		}
	}
}

func TestStoreImageBytes_EncodeFailureFallsBack(t *testing.T) {
	// Empty PATH so cwebp lookup fails; storeImageBytes should still succeed
	// with HasWebP=false and URL pointing at the original.
	t.Setenv("PATH", "")
	chdirTo(t, t.TempDir())
	t.Setenv("DEV", "true")

	pngBytes := tinyImageBytes(t, func(w *bytes.Buffer, img image.Image) error { return png.Encode(w, img) })
	url, meta, err := storeImageBytes("fallback-id", pngBytes)
	if err != nil {
		t.Fatalf("storeImageBytes: %v", err)
	}
	if url != "/images/fallback-id.png" {
		t.Fatalf("url: want /images/fallback-id.png, got %s", url)
	}
	if meta.HasWebP {
		t.Fatal("expected HasWebP=false when cwebp missing")
	}
	if meta.FallbackExt != "png" {
		t.Fatalf("fallbackExt: want png, got %s", meta.FallbackExt)
	}
	// Dimensions still read via stdlib even without cwebp.
	if meta.Width != 32 || meta.Height != 24 {
		t.Fatalf("dims: got %dx%d", meta.Width, meta.Height)
	}
	// No .webp file expected.
	if _, err := os.Stat("images/fallback-id.webp"); !os.IsNotExist(err) {
		t.Fatalf("unexpected .webp: err=%v", err)
	}
}

func TestReadImageMeta_MissingReturnsZero(t *testing.T) {
	chdirTo(t, t.TempDir())
	meta, err := readImageMeta("nonexistent-id")
	if err != nil {
		t.Fatalf("readImageMeta: %v", err)
	}
	if meta != (ImageMeta{}) {
		t.Fatalf("expected zero meta, got %+v", meta)
	}
}

// Sanity: generated fixture is the expected size.
func TestFixture_SizeSanity(t *testing.T) {
	pngBytes := tinyImageBytes(t, func(w *bytes.Buffer, img image.Image) error { return png.Encode(w, img) })
	cfg, _, err := image.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 32 || cfg.Height != 24 {
		t.Fatalf("fixture dims: %dx%d", cfg.Width, cfg.Height)
	}
	// Avoid unused-filepath-import lint.
	_ = filepath.Join("", "")
}
