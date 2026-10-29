package captureflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pipelinedImaging struct {
	*fakeImaging
}

func (p pipelinedImaging) InspectCapture(
	ctx context.Context,
	req *kernelports.ImagingCaptureRequest,
	yield func(*kernelports.ImagingCaptureResult) error,
) (*kernelports.ImagingCaptureSummary, error) {
	sent := make(chan int)
	go func() {
		defer close(sent)
		for page, err := range req.Pages() {
			if err != nil {
				return
			}
			if _, err = readSource(ctx, page.Source); err != nil {
				return
			}
			sent <- page.Index
		}
	}()
	inspected := 0
	for index := range sent {
		if err := yield(&kernelports.ImagingCaptureResult{
			Index:                index,
			Width:                1275,
			Height:               1650,
			Thumbnail:            []byte("thumbnail"),
			ThumbnailContentType: "image/jpeg",
			InkCoverage:          0.05,
		}); err != nil {
			return nil, err
		}
		inspected++
	}
	return &kernelports.ImagingCaptureSummary{Inspected: inspected}, nil
}

func TestReadPages_PagesStorageCouldNotReadFailWhileTheBatchStreams(t *testing.T) {
	t.Parallel()

	w := newWorld()
	svc := w.service()
	svc.imaging = pipelinedImaging{&fakeImaging{w}}

	const count = 12
	pages := make([]*capture.CapturePage, 0, count)
	pending := make([]int, 0, count)
	for i := range count {
		page := &capture.CapturePage{
			ID:          pulid.MustNew("cpg_"),
			Status:      capture.PageReceived,
			StoragePath: fmt.Sprintf("org/capture/pages/%d.pdf", i),
		}
		if i%3 != 0 {
			require.NoError(t, svc.support().PutObject(
				t.Context(),
				w.tenant,
				page.StoragePath,
				"application/pdf",
				pdfPage(t, uint8(i)),
			))
		}
		pages = append(pages, page)
		pending = append(pending, i)
	}

	done := 0
	err := svc.processing().ReadPages(
		t.Context(),
		w.tenant,
		pages,
		pending,
		kernelports.ImagingBatch,
		func(int) { done++ },
	)

	require.NoError(t, err)
	assert.Equal(t, count, done)
	for i, page := range pages {
		if i%3 == 0 {
			assert.Equal(t, capture.PageFailed, page.Status, "page %d", i)
			assert.Equal(t, "The page could not be read back from storage.", page.FailureMessage)
			continue
		}
		assert.NotEqual(t, capture.PageFailed, page.Status, "page %d", i)
	}
}
