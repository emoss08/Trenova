// Package captureimaging reads captured pages: it renders a one-page PDF,
// makes the thumbnail the intake queue shows, measures how much of the page
// carries ink, and decodes any QR codes on it.
package captureimaging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/fitzdoc"
	"github.com/makiuchi-d/gozxing"
	multiqr "github.com/makiuchi-d/gozxing/multi/qrcode"
)

const (
	// renderDPI is enough for a cover sheet's QR code to decode from a skewed
	// scan and for ink coverage to be measured, at a quarter of the pixels a
	// 300 DPI render would cost.
	renderDPI = 150
	// thumbnailWidth matches the page strip in the intake queue at 2x.
	thumbnailWidth   = 240
	thumbnailQuality = 72
	// inkThreshold is the luminance below which a pixel counts as ink. Scanner
	// noise and paper tint sit well above it.
	inkThreshold = 160
	// marginFraction is left out of the ink measure on every side: scanners
	// leave shadows and punch holes at the edges of blank sheets.
	marginFraction = 0.06
	// sampleStride skips pixels when measuring ink. A blank page is blank at
	// every other pixel too, and a page with a signature still has hundreds of
	// sampled dark pixels.
	sampleStride = 2
)

var ErrNoPages = errors.New("the PDF has no pages")

type Inspector struct{}

func New() services.CapturePageInspector {
	return &Inspector{}
}

func (i *Inspector) Inspect(
	ctx context.Context,
	pdf []byte,
) (*services.CapturePageInspection, error) {
	if !fitzdoc.Available() {
		return nil, services.ErrPageInspectionUnavailable
	}

	doc, err := fitzdoc.NewFromMemory(pdf)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	if doc.NumPage() < 1 {
		return nil, ErrNoPages
	}

	if err = ctx.Err(); err != nil {
		return nil, err
	}

	rendered, err := doc.ImageDPI(0, renderDPI)
	if err != nil {
		return nil, fmt.Errorf("render page: %w", err)
	}

	return InspectImage(ctx, rendered)
}

// InspectImage reads an already rendered page. It is separate so the reading
// can be tested without a PDF renderer.
func InspectImage(ctx context.Context, img image.Image) (*services.CapturePageInspection, error) {
	bounds := img.Bounds()

	thumbnail, err := encodeThumbnail(img)
	if err != nil {
		return nil, err
	}

	if err = ctx.Err(); err != nil {
		return nil, err
	}

	return &services.CapturePageInspection{
		WidthPx:     bounds.Dx(),
		HeightPx:    bounds.Dy(),
		Thumbnail:   thumbnail,
		InkCoverage: InkCoverage(img),
		Codes:       DecodeQRCodes(img),
	}, nil
}

func encodeThumbnail(img image.Image) ([]byte, error) {
	resized := imaging.Resize(img, thumbnailWidth, 0, imaging.Lanczos)

	out := new(bytes.Buffer)
	if err := jpeg.Encode(out, resized, &jpeg.Options{Quality: thumbnailQuality}); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}

	return out.Bytes(), nil
}

// InkCoverage is the share of sampled pixels inside the margins that are dark
// enough to be content.
func InkCoverage(img image.Image) float64 {
	bounds := img.Bounds()
	marginX := int(float64(bounds.Dx()) * marginFraction)
	marginY := int(float64(bounds.Dy()) * marginFraction)

	minX, maxX := bounds.Min.X+marginX, bounds.Max.X-marginX
	minY, maxY := bounds.Min.Y+marginY, bounds.Max.Y-marginY
	if minX >= maxX || minY >= maxY {
		return 0
	}

	var sampled, dark int
	if rgba, ok := img.(*image.RGBA); ok {
		for y := minY; y < maxY; y += sampleStride {
			row := rgba.PixOffset(0, y)
			for x := minX; x < maxX; x += sampleStride {
				p := row + (x-bounds.Min.X)*4
				r, g, b := uint32(rgba.Pix[p]), uint32(rgba.Pix[p+1]), uint32(rgba.Pix[p+2])
				if (299*r+587*g+114*b)/1000 < inkThreshold {
					dark++
				}
				sampled++
			}
		}
	} else {
		for y := minY; y < maxY; y += sampleStride {
			for x := minX; x < maxX; x += sampleStride {
				r, g, b, _ := img.At(x, y).RGBA()
				if (299*r+587*g+114*b)/1000>>8 < inkThreshold {
					dark++
				}
				sampled++
			}
		}
	}

	if sampled == 0 {
		return 0
	}

	return float64(dark) / float64(sampled)
}

// DecodeQRCodes returns every QR payload on the page. A page with none, or one
// the decoder cannot read, returns nothing rather than an error: most pages
// carry no code, and an unreadable code is a page to route by hand.
func DecodeQRCodes(img image.Image) []string {
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil
	}

	hints := make(map[gozxing.DecodeHintType]any, 1)
	hints[gozxing.DecodeHintType_TRY_HARDER] = true

	results, err := multiqr.NewQRCodeMultiReader().DecodeMultiple(bitmap, hints)
	if err != nil {
		return nil
	}

	codes := make([]string, 0, len(results))
	for _, result := range results {
		if text := result.GetText(); text != "" {
			codes = append(codes, text)
		}
	}

	return codes
}
