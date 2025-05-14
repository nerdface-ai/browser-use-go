package browser

import (
	"runtime"

	"github.com/kbinani/screenshot"
)

// getScreenResolution returns the width and height of the main (first) display
func getScreenResolution() map[string]int {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return map[string]int{"width": 1920, "height": 1080} // fallback
	}
	b := screenshot.GetDisplayBounds(0)
	return map[string]int{
		"width":  b.Dx(),
		"height": b.Dy(),
	}
}

// getWindowAdjustments returns (borderAdjust, titlebarAdjust) for major OSes
func getWindowAdjustments() (int, int) {
	os := runtime.GOOS
	switch os {
	case "darwin":
		return -4, 24 // macOS: small title bar, no border
	case "win32":
		return -8, 0 // Windows: border on the left
	default:
		return 0, 0 // Linux or others
	}
}
