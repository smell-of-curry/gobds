package skinutil

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/player/skin"
)

// skinLayout ...
type skinLayout struct {
	headOffset    image.Point
	overlayOffset image.Point
	headSize      int
	hasOverlay    bool
}

const (
	headSize = 8
)

var (
	skinLayouts = map[image.Point]skinLayout{
		{64, 32}: {
			headOffset: image.Point{X: 8, Y: 8},
			headSize:   headSize,
			hasOverlay: false,
		},
		{64, 64}: {
			headOffset:    image.Point{X: 8, Y: 8},
			overlayOffset: image.Point{X: 40, Y: 8},
			headSize:      headSize,
			hasOverlay:    true,
		},
		{128, 128}: {
			headOffset:    image.Point{X: 16, Y: 16},
			overlayOffset: image.Point{X: 80, Y: 16},
			headSize:      headSize * 2,
			hasOverlay:    true,
		},
	}
)

// ExtractHead ...
func ExtractHead(s skin.Skin) image.Image {
	bounds := s.Bounds()
	size := image.Point{X: bounds.Dx(), Y: bounds.Dy()}

	layout, exists := skinLayouts[size]
	if !exists {
		return image.NewRGBA(image.Rect(0, 0, headSize, headSize))
	}

	head := image.NewRGBA(image.Rect(0, 0, layout.headSize, layout.headSize))
	faceRect := image.Rect(
		layout.headOffset.X,
		layout.headOffset.Y,
		layout.headOffset.X+layout.headSize,
		layout.headOffset.Y+layout.headSize,
	)
	draw.Draw(head, head.Bounds(), &s, faceRect.Min, draw.Src)

	if layout.hasOverlay {
		overlayRect := image.Rect(
			layout.overlayOffset.X,
			layout.overlayOffset.Y,
			layout.overlayOffset.X+layout.headSize,
			layout.overlayOffset.Y+layout.headSize,
		)
		draw.Draw(head, head.Bounds(), &s, overlayRect.Min, draw.Over)
	}

	if layout.headSize > headSize {
		return scaleDown(head, headSize)
	}
	return head
}

// scaleDown ...
func scaleDown(src image.Image, targetSize int) image.Image {
	srcBounds := src.Bounds()
	srcSize := srcBounds.Dx()
	scale := srcSize / targetSize

	dst := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
	for y := 0; y < targetSize; y++ {
		for x := 0; x < targetSize; x++ {
			srcX := x * scale
			srcY := y * scale
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// SaveHeadPNG ...
func SaveHeadPNG(xuid string, head image.Image, dir string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("create heads directory: %w", err)
	}

	path := filepath.Join(dir, xuid+".png")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create head file: %w", err)
	}
	defer file.Close()

	if err = png.Encode(file, head); err != nil {
		return fmt.Errorf("encode head png: %w", err)
	}
	return nil
}
