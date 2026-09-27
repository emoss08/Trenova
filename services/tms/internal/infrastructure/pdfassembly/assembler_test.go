package pdfassembly_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/pdfassembly"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func onePagePDF(t *testing.T, shade uint8) []byte {
	t.Helper()

	img := image.NewGray(image.Rect(0, 0, 60, 80))
	for i := range img.Pix {
		img.Pix[i] = shade
	}
	img.Set(5, 5, color.Black)

	encoded := new(bytes.Buffer)
	require.NoError(t, png.Encode(encoded, img))

	out := new(bytes.Buffer)
	imp := pdfcpu.DefaultImportConfig()
	require.NoError(t, api.ImportImages(nil, out, []io.Reader{encoded}, imp, model.NewDefaultConfiguration()))

	return out.Bytes()
}

func TestAssembleSplitRoundTrip(t *testing.T) {
	t.Parallel()

	assembler := pdfassembly.New()
	pages := []services.CaptureAssemblyPage{
		{PDF: onePagePDF(t, 250)},
		{PDF: onePagePDF(t, 200), Rotation: 90},
		{PDF: onePagePDF(t, 150), Rotation: -90},
	}

	joined, err := assembler.Assemble(t.Context(), pages)
	require.NoError(t, err)

	count, err := assembler.PageCount(t.Context(), joined)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	split, err := assembler.Split(t.Context(), joined)
	require.NoError(t, err)
	require.Len(t, split, 3)
	for _, page := range split {
		n, countErr := assembler.PageCount(t.Context(), page)
		require.NoError(t, countErr)
		assert.Equal(t, 1, n)
	}
}

func TestAssembleSinglePageIsUntouched(t *testing.T) {
	t.Parallel()

	page := onePagePDF(t, 240)
	joined, err := pdfassembly.New().Assemble(t.Context(), []services.CaptureAssemblyPage{{PDF: page}})
	require.NoError(t, err)
	assert.Equal(t, page, joined, "an unedited page keeps its original bytes")
}

func TestRejectsGarbage(t *testing.T) {
	t.Parallel()

	assembler := pdfassembly.New()
	_, err := assembler.PageCount(t.Context(), []byte("%PDF-1.7 not really"))
	require.ErrorIs(t, err, pdfassembly.ErrUnreadable)

	_, err = assembler.Assemble(t.Context(), nil)
	require.ErrorIs(t, err, pdfassembly.ErrNoPages)
}
