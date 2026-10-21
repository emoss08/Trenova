package thumbnailservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func source() kernelports.ImagingSource {
	return kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7"))
}

func TestSupportsThumbnail(t *testing.T) {
	t.Parallel()

	g := thumbnailservice.NewGenerator(imagingfake.New())
	assert.True(t, g.SupportsThumbnail("application/pdf"))
	assert.True(t, g.SupportsThumbnail("IMAGE/HEIC"))
	assert.False(t, g.SupportsThumbnail("text/plain"))
}

func TestGenerateAsksForTheFirstPageAsAThumbnail(t *testing.T) {
	t.Parallel()

	fake := imagingfake.New()
	var asked *kernelports.ImagingRenderRequest
	base := fake.Render
	fake.RenderFunc = func(
		ctx context.Context,
		req *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		asked = req
		fake.RenderFunc = nil
		defer func() { fake.RenderFunc = nil }()
		return base(ctx, req, yield)
	}

	data, err := thumbnailservice.NewGenerator(fake).Generate(t.Context(), source(), kernelports.ImagingBatch)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	require.NotNil(t, asked)
	assert.Equal(t, []int{0}, asked.Pages)
	assert.Equal(t, thumbnailservice.ThumbnailWidth, asked.FitWidth)
	assert.Equal(t, kernelports.ImagingWebP, asked.Format)
	assert.Equal(t, kernelports.ImagingBatch, asked.Priority)
}

func TestRenderPageMapsAMissingPage(t *testing.T) {
	t.Parallel()

	fake := imagingfake.New()
	fake.PageCount = 2
	g := thumbnailservice.NewGenerator(fake)

	data, err := g.RenderPage(t.Context(), source(), 2, kernelports.ImagingInteractive)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = g.RenderPage(t.Context(), source(), 3, kernelports.ImagingInteractive)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)

	_, err = g.RenderPage(t.Context(), source(), 0, kernelports.ImagingInteractive)
	require.ErrorIs(t, err, thumbnailservice.ErrPageOutOfRange)
}

func TestAnEmptyDocumentHasNoThumbnail(t *testing.T) {
	t.Parallel()

	fake := imagingfake.New()
	fake.PageCount = 0
	_, err := thumbnailservice.NewGenerator(fake).Generate(t.Context(), source(), kernelports.ImagingBatch)
	require.ErrorIs(t, err, thumbnailservice.ErrPDFHasNoPages)
}

func TestAnUnreadableDocumentIsReportedAsSuch(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{
		kernelports.ErrImagingInvalidDocument,
		kernelports.ErrImagingEncrypted,
		kernelports.ErrImagingPixelBudget,
	} {
		fake := imagingfake.New()
		fake.RenderFunc = func(
			context.Context,
			*kernelports.ImagingRenderRequest,
			func(*kernelports.ImagingArtifact) error,
		) (*kernelports.ImagingRenderSummary, error) {
			return nil, fmt.Errorf("%w: from the worker", cause)
		}
		_, err := thumbnailservice.NewGenerator(fake).Generate(t.Context(), source(), kernelports.ImagingBatch)
		require.ErrorIs(t, err, thumbnailservice.ErrUnreadable, cause.Error())
		require.ErrorIs(t, err, cause)
	}
}

func TestRenderPagesYieldsEveryPageNumberedFromOne(t *testing.T) {
	t.Parallel()

	fake := imagingfake.New()
	fake.PageCount = 3
	var pages []int
	count, err := thumbnailservice.NewGenerator(fake).RenderPages(
		t.Context(), source(), 20, kernelports.ImagingBatch,
		func(page int, data []byte) error {
			assert.NotEmpty(t, data)
			pages = append(pages, page)
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, []int{1, 2, 3}, pages)
}

func TestWithoutTheImagingServiceNothingRenders(t *testing.T) {
	t.Parallel()

	_, err := thumbnailservice.NewGenerator(nil).Generate(t.Context(), source(), kernelports.ImagingBatch)
	require.ErrorIs(t, err, thumbnailservice.ErrImagingNotConfigured)
}
