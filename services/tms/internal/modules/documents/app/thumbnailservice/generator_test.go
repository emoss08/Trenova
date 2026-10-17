package thumbnailservice_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/pdftest"
	"github.com/gen2brain/webp"
	"github.com/stretchr/testify/require"
)

var minimalPDF = []byte(`%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>endobj
trailer<</Root 1 0 R>>
`)

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for x := range 640 {
		for y := range 480 {
			img.Set(x, y, color.RGBA{R: 30, G: 120, B: 200, A: 255})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func requireWebP(t *testing.T, data []byte) {
	t.Helper()

	require.Greater(t, len(data), 12)
	require.Equal(t, "RIFF", string(data[:4]))
	require.Equal(t, "WEBP", string(data[8:12]))
}

func TestGenerateFromImage(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	thumb, err := generator.Generate(t.Context(), bytes.NewReader(testPNG(t)), "image/png")
	require.NoError(t, err)
	requireWebP(t, thumb)
}

func TestGenerateFromImageInvalidData(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	_, err := generator.Generate(t.Context(), bytes.NewReader([]byte("not an image")), "image/png")
	require.ErrorContains(t, err, "failed to decode image")
}

func TestGenerateFromPDF(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	thumb, err := generator.Generate(t.Context(), bytes.NewReader(minimalPDF), "application/pdf")
	require.NoError(t, err)
	requireWebP(t, thumb)
}

func TestGenerateFromPDFInvalidData(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	_, err := generator.Generate(
		t.Context(),
		bytes.NewReader([]byte("not a pdf")),
		"application/pdf",
	)
	require.ErrorIs(t, err, kernelports.ErrPDFUnreadable)
	require.ErrorContains(t, err, "failed to open PDF")
}

func TestGenerateFromPDFWithoutReader(t *testing.T) {
	generator := thumbnailservice.NewGenerator(nil)

	_, err := generator.Generate(t.Context(), bytes.NewReader(minimalPDF), "application/pdf")
	require.ErrorIs(t, err, thumbnailservice.ErrPDFReaderUnavailable)
}

func TestGenerateUnsupportedContentType(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	_, err := generator.Generate(t.Context(), bytes.NewReader(nil), "text/plain")
	require.ErrorContains(t, err, "unsupported content type")
}

func TestSupportsThumbnail(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	require.True(t, generator.SupportsThumbnail("image/png"))
	require.True(t, generator.SupportsThumbnail("IMAGE/JPEG"))
	require.True(t, generator.SupportsThumbnail("application/pdf"))
	require.False(t, generator.SupportsThumbnail("text/plain"))
	require.False(t, generator.SupportsThumbnail("application/zip"))
}

var twoPagePDF = []byte(`%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R 4 0 R]/Count 2>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>endobj
4 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 792 612]>>endobj
trailer<</Root 1 0 R>>
`)

func webPSize(t *testing.T, data []byte) (int, int) {
	t.Helper()

	requireWebP(t, data)
	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return cfg.Width, cfg.Height
}

func TestRenderPageDrawsTheRequestedPDFPage(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	first, err := generator.RenderPage(
		t.Context(), bytes.NewReader(twoPagePDF), "application/pdf", 1,
	)
	require.NoError(t, err)
	width, height := webPSize(t, first)
	require.InDelta(t, 1275, width, 1)
	require.InDelta(t, 1650, height, 1)

	second, err := generator.RenderPage(
		t.Context(), bytes.NewReader(twoPagePDF), "application/pdf", 2,
	)
	require.NoError(t, err)
	width, height = webPSize(t, second)
	require.Equal(t, thumbnailservice.PageMaxWidth, width)
	require.InDelta(t, 1081, height, 1)
}

func TestRenderPagePastTheLastPDFPage(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	_, err := generator.RenderPage(
		t.Context(), bytes.NewReader(twoPagePDF), "application/pdf", 3,
	)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)

	_, err = generator.RenderPage(
		t.Context(), bytes.NewReader(twoPagePDF), "application/pdf", 0,
	)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)
}

func TestRenderPageKeepsAnImageAtItsOwnSize(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	page, err := generator.RenderPage(t.Context(), bytes.NewReader(testPNG(t)), "image/png", 1)
	require.NoError(t, err)
	width, height := webPSize(t, page)
	require.Equal(t, 640, width)
	require.Equal(t, 480, height)
}

func TestRenderPageAnImageHasOnePage(t *testing.T) {
	generator := thumbnailservice.NewGenerator(pdftest.Reader())

	_, err := generator.RenderPage(t.Context(), bytes.NewReader(testPNG(t)), "image/png", 2)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)
}
