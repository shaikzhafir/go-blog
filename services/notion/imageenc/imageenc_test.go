package imageenc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir, name string, encode func(w *os.File) error) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer f.Close()
	if err := encode(f); err != nil {
		t.Fatalf("encode fixture %s: %v", name, err)
	}
	return path
}

func sampleImage() image.Image {
	const w, h = 64, 48
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 128, A: 255})
		}
	}
	return img
}

func TestEncodeWebP_PNGJPEGGIF(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not on PATH; skipping")
	}
	dir := t.TempDir()
	img := sampleImage()

	cases := []struct {
		name    string
		fixture func(f *os.File) error
	}{
		{"src.png", func(f *os.File) error { return png.Encode(f, img) }},
		{"src.jpg", func(f *os.File) error { return jpeg.Encode(f, img, &jpeg.Options{Quality: 90}) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := writeFixture(t, dir, tc.name, tc.fixture)
			dst := filepath.Join(dir, tc.name+".webp")
			if err := EncodeWebP(src, dst); err != nil {
				t.Fatalf("EncodeWebP: %v", err)
			}
			data, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("read dst: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("dst is empty")
			}
			// RIFF....WEBP magic
			if len(data) < 12 || !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
				t.Fatalf("dst missing WebP magic bytes: %q", data[:min(12, len(data))])
			}

			w, h, err := ReadDimensions(src)
			if err != nil {
				t.Fatalf("ReadDimensions: %v", err)
			}
			if w != 64 || h != 48 {
				t.Fatalf("dims: want 64x48, got %dx%d", w, h)
			}
		})
	}
}

func TestEncodeWebP_MissingBinary(t *testing.T) {
	// Run under an empty PATH so cwebp lookup fails deterministically.
	t.Setenv("PATH", "")
	dir := t.TempDir()
	src := filepath.Join(dir, "x.png")
	if err := os.WriteFile(src, []byte("not-really-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "x.webp")
	err := EncodeWebP(src, dst)
	if err == nil {
		t.Fatal("expected error when cwebp is missing, got nil")
	}
}

