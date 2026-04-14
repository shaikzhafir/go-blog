// Package imageenc wraps the `cwebp` binary and stdlib image decoders so the
// rest of the codebase can stay free of cgo and WebP-specific Go deps.
package imageenc

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
)

// Quality is the fixed cwebp -q value. 85 matches the design decision.
const Quality = 85

// EncodeWebP shells out to `cwebp -q 85 -quiet -o dstPath srcPath`.
// Returns an error with combined stderr when cwebp is missing or exits non-zero.
func EncodeWebP(srcPath, dstPath string) error {
	cmd := exec.Command("cwebp", "-q", fmt.Sprintf("%d", Quality), "-quiet", "-o", dstPath, srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cwebp %s -> %s: %w (output: %s)", srcPath, dstPath, err, string(out))
	}
	return nil
}

// ReadDimensions returns the intrinsic pixel width and height of an image file.
// Supports PNG, JPEG, GIF via stdlib decoders registered via blank imports.
func ReadDimensions(srcPath string) (int, int, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// Available reports whether the cwebp binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("cwebp")
	return err == nil
}
