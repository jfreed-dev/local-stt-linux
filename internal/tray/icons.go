package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

const iconSize = 22

var (
	iconIdle       []byte
	iconRecording  []byte
	iconError      []byte
)

func init() {
	iconIdle = generateIcon(color.RGBA{160, 160, 160, 255}, false)      // gray mic
	iconRecording = generateIcon(color.RGBA{46, 204, 113, 255}, true)   // green mic + dot
	iconError = generateIcon(color.RGBA{231, 76, 60, 255}, false)       // red mic
}

// generateIcon creates a simple microphone icon as PNG bytes.
func generateIcon(col color.RGBA, recording bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))

	// Transparent background
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	cx := float64(iconSize) / 2
	drawMic(img, cx, col)

	// Recording indicator: small filled circle in top-right
	if recording {
		dot := color.RGBA{231, 76, 60, 255} // red dot
		dcx, dcy := float64(iconSize-4), float64(4)
		for y := 0; y < iconSize; y++ {
			for x := 0; x < iconSize; x++ {
				dx := float64(x) - dcx
				dy := float64(y) - dcy
				if dx*dx+dy*dy <= 3*3 {
					img.Set(x, y, dot)
				}
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// drawMic draws a simple microphone shape centered at cx.
func drawMic(img *image.RGBA, cx float64, col color.RGBA) {
	// Mic head: rounded rectangle / capsule shape (top portion)
	headTop := 1.0
	headBot := 10.0
	headW := 3.5

	for y := int(headTop); y <= int(headBot); y++ {
		for x := 0; x < iconSize; x++ {
			dx := float64(x) - cx
			fy := float64(y)

			// Capsule: rectangle with rounded top and bottom
			inRect := math.Abs(dx) <= headW
			// Round the top
			if fy < headTop+headW {
				dy := fy - (headTop + headW)
				inRect = dx*dx+dy*dy <= headW*headW
			}
			// Round the bottom
			if fy > headBot-headW {
				dy := fy - (headBot - headW)
				inRect = dx*dx+dy*dy <= headW*headW
			}

			if inRect {
				img.Set(x, y, col)
			}
		}
	}

	// Mic body arc (U-shape around the head)
	arcCy := 8.0
	arcR := 5.5
	for y := int(arcCy); y <= int(arcCy+arcR)+1; y++ {
		for x := 0; x < iconSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - arcCy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= arcR-0.8 && dist <= arcR+0.8 && dy >= 0 {
				img.Set(x, y, col)
			}
		}
	}

	// Stand: vertical line from bottom of arc
	standTop := int(arcCy + arcR)
	standBot := 18
	for y := standTop; y <= standBot; y++ {
		img.Set(int(cx), y, col)
		img.Set(int(cx)-1, y, col)
	}

	// Base: horizontal line at bottom
	baseY := standBot + 1
	for x := int(cx) - 4; x <= int(cx)+4; x++ {
		img.Set(x, baseY, col)
		img.Set(x, baseY+1, col)
	}
}
