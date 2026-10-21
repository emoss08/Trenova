//go:build integration

package thumbnailservice_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/imagingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

func TestThumbnailsAndPageViewsFromTheWorker(t *testing.T) {
	g := thumbnailservice.NewGenerator(imagingtest.Client(t))
	pdf := kernelports.ImagingBytes("application/pdf", imagingtest.Fixture(t, "rate-confirmation.pdf"))

	thumb, err := g.Generate(t.Context(), pdf, kernelports.ImagingBatch)
	require.NoError(t, err)
	decoded, _, err := image.Decode(bytes.NewReader(thumb))
	require.NoError(t, err)
	assert.Equal(t, thumbnailservice.ThumbnailWidth, decoded.Bounds().Dx())

	view, err := g.RenderPage(t.Context(), pdf, 2, kernelports.ImagingInteractive)
	require.NoError(t, err)
	decoded, _, err = image.Decode(bytes.NewReader(view))
	require.NoError(t, err)
	assert.Equal(t, thumbnailservice.PageMaxHeight, decoded.Bounds().Dy())

	_, err = g.RenderPage(t.Context(), pdf, 3, kernelports.ImagingInteractive)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)

	_, err = g.Generate(t.Context(),
		kernelports.ImagingBytes("application/pdf", imagingtest.Fixture(t, "hostile-garbage.pdf")),
		kernelports.ImagingBatch)
	require.ErrorIs(t, err, thumbnailservice.ErrUnreadable)

	pages := 0
	count, err := g.RenderPages(t.Context(),
		kernelports.ImagingBytes("application/pdf", imagingtest.Fixture(t, "native-10p.pdf")), 4,
		kernelports.ImagingBatch, func(int, []byte) error {
			pages++
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, 10, count)
	assert.Equal(t, 4, pages, "pre-rendering stops at its page limit")
}
