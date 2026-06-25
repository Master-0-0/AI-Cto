//go:build !windows

package implant

import "syscall"

func hideWindow() *syscall.SysProcAttr {
	return nil // No-op on non-Windows platforms
}
