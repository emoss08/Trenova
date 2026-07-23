package thumbnailservice_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
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
	generator := thumbnailservice.NewGenerator()

	thumb, err := generator.Generate(t.Context(), bytes.NewReader(testPNG(t)), "image/png")
	require.NoError(t, err)
	requireWebP(t, thumb)
}

func TestGenerateFromImageInvalidData(t *testing.T) {
	generator := thumbnailservice.NewGenerator()

	_, err := generator.Generate(t.Context(), bytes.NewReader([]byte("not an image")), "image/png")
	require.ErrorContains(t, err, "failed to decode image")
}

func TestGenerateFromPDFInProcess(t *testing.T) {
	generator := thumbnailservice.NewInProcessGenerator()

	thumb, err := generator.Generate(t.Context(), bytes.NewReader(minimalPDF), "application/pdf")
	require.NoError(t, err)
	requireWebP(t, thumb)
}

func TestGenerateFromPDFInProcessInvalidData(t *testing.T) {
	generator := thumbnailservice.NewInProcessGenerator()

	_, err := generator.Generate(
		t.Context(),
		bytes.NewReader([]byte("not a pdf")),
		"application/pdf",
	)
	require.ErrorContains(t, err, "failed to open PDF")
}

func TestGenerateUnsupportedContentType(t *testing.T) {
	generator := thumbnailservice.NewGenerator()

	_, err := generator.Generate(t.Context(), bytes.NewReader(nil), "text/plain")
	require.ErrorContains(t, err, "unsupported content type")
}

func TestSupportsThumbnail(t *testing.T) {
	generator := thumbnailservice.NewGenerator()

	require.True(t, generator.SupportsThumbnail("image/png"))
	require.True(t, generator.SupportsThumbnail("IMAGE/JPEG"))
	require.True(t, generator.SupportsThumbnail("application/pdf"))
	require.False(t, generator.SupportsThumbnail("text/plain"))
	require.False(t, generator.SupportsThumbnail("application/zip"))
}
