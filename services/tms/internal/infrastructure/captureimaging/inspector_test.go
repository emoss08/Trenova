package captureimaging_test

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/infrastructure/captureimaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pageWidth, pageHeight = 1275, 1650

func blankPage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pageWidth, pageHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	return img
}

func TestBlankPageHasNoInk(t *testing.T) {
	t.Parallel()

	page := blankPage()
	// A punch-hole shadow in the margin is not content.
	draw.Draw(page, image.Rect(10, 10, 60, 60), image.NewUniform(color.Black), image.Point{}, draw.Src)

	coverage := captureimaging.InkCoverage(page)
	assert.Less(t, coverage, capture.BlankThreshold)
}

func TestSignatureIsNotBlank(t *testing.T) {
	t.Parallel()

	page := blankPage()
	draw.Draw(page, image.Rect(400, 1200, 800, 1260), image.NewUniform(color.Black), image.Point{}, draw.Src)

	assert.Greater(t, captureimaging.InkCoverage(page), capture.BlankThreshold)
	assert.InDelta(t,
		captureimaging.InkCoverage(page),
		captureimaging.InkCoverage(page.SubImage(page.Bounds())),
		0.0001,
		"the fast and generic paths must agree")
}

func TestDecodesCoverSheetQRCode(t *testing.T) {
	t.Parallel()

	payload := capture.CoverSheetPayload("kq3Xw8mZ-token")
	matrix, err := qrcode.NewQRCodeWriter().Encode(payload, gozxing.BarcodeFormat_QR_CODE, 400, 400, nil)
	require.NoError(t, err)

	page := blankPage()
	for y := range 400 {
		for x := range 400 {
			if matrix.Get(x, y) {
				page.Set(300+x, 200+y, color.Black)
			}
		}
	}

	inspection, err := captureimaging.InspectImage(t.Context(), page)
	require.NoError(t, err)
	assert.Equal(t, []string{payload}, inspection.Codes)
	assert.Equal(t, pageWidth, inspection.WidthPx)
	assert.NotEmpty(t, inspection.Thumbnail)
	assert.Greater(t, inspection.InkCoverage, capture.BlankThreshold)
}

func TestPageWithoutCodes(t *testing.T) {
	t.Parallel()

	assert.Empty(t, captureimaging.DecodeQRCodes(blankPage()))
}
