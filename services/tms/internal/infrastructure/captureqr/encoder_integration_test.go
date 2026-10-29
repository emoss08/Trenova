//go:build integration

package captureqr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"iter"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/imagingtest"
	"github.com/stretchr/testify/require"
)

func render(t *testing.T, modules []string) []byte {
	t.Helper()

	const scale, quiet = 8, 4
	side := (len(modules) + 2*quiet) * scale
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

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestTheWorkerReadsAnEncodedSheetBackAsItsPayload(t *testing.T) {
	payload := capturecoversheet.CoverSheetPayload("dGhpcy1pcy1hLXJhbmRvbS10b2tlbi1mb3ItdGVzdGluZw")
	code, err := New().Encode(payload)
	require.NoError(t, err)

	source := kernelports.ImagingBytes("image/png", render(t, code.Modules))
	var read []string
	_, err = imagingtest.Client(t).InspectCapture(t.Context(), &kernelports.ImagingCaptureRequest{
		PageCount: 1,
		Pages: func() iter.Seq2[kernelports.ImagingCapturePage, error] {
			return func(yield func(kernelports.ImagingCapturePage, error) bool) {
				yield(kernelports.ImagingCapturePage{Source: source}, nil)
			}
		},
	}, func(result *kernelports.ImagingCaptureResult) error {
		if result.Err != nil {
			return result.Err
		}
		for _, barcode := range result.Codes {
			read = append(read, barcode.Text)
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{payload}, read)
}
