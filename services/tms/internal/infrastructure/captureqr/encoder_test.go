package captureqr

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// render draws the modules the way a printed sheet would, with the quiet zone
// the encoder leaves to its caller.
func render(t *testing.T, modules []string) image.Image {
	t.Helper()

	const scale, quiet = 8, 4
	size := len(modules)
	side := (size + 2*quiet) * scale
	img := image.NewGray(image.Rect(0, 0, side, side))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	for y, row := range modules {
		for x, module := range row {
			if module != '1' {
				continue
			}
			for dy := range scale {
				for dx := range scale {
					img.Set((x+quiet)*scale+dx, (y+quiet)*scale+dy, color.Black)
				}
			}
		}
	}

	return img
}

func TestEncodedSheetReadsBackAsItsPayload(t *testing.T) {
	t.Parallel()

	payload := capture.CoverSheetPayload("dGhpcy1pcy1hLXJhbmRvbS10b2tlbi1mb3ItdGVzdGluZw")
	code, err := New().Encode(payload)
	require.NoError(t, err)
	require.Equal(t, code.Size, len(code.Modules))
	for _, row := range code.Modules {
		require.Len(t, row, code.Size)
		assert.Empty(t, strings.Trim(row, "01"))
	}

	bitmap, err := gozxing.NewBinaryBitmapFromImage(render(t, code.Modules))
	require.NoError(t, err)
	result, err := qrcode.NewQRCodeReader().Decode(bitmap, nil)
	require.NoError(t, err)
	assert.Equal(t, payload, result.GetText())
}

func TestEncodeRefusesAnEmptyOrOversizedPayload(t *testing.T) {
	t.Parallel()

	_, err := New().Encode("")
	require.ErrorIs(t, err, ErrPayloadInvalid)

	_, err = New().Encode(strings.Repeat("x", maxPayloadLength+1))
	require.ErrorIs(t, err, ErrPayloadInvalid)
}
