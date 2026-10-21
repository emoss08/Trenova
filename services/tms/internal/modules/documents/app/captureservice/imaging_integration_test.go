//go:build integration

package captureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/testutil/imagingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheWorkerReadsAScannedBatch(t *testing.T) {
	w := newWorld()
	s := w.service()
	s.imaging = imagingtest.Client(t)
	principal := principalFor(t, s, pair(t, w, s))

	pages := [][]byte{
		imagingtest.Fixture(t, "capture-text-0.pdf"),
		imagingtest.Fixture(t, "capture-cover-2qr.pdf"),
		imagingtest.Fixture(t, "capture-blank.pdf"),
	}
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "real", Source: capture.SourceScan}, pages, nil)

	_, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)

	read := w.pagesOf(batch.ID)
	require.Len(t, read, 3)
	for _, page := range read {
		assert.Equal(t, capture.PageProcessed, page.Status, page.FailureMessage)
		assert.NotEmpty(t, page.ThumbnailPath)
		require.NotNil(t, page.BlankScore)
	}
	assert.Greater(t, *read[0].BlankScore, 0.01, "a page of text carries ink")
	assert.ElementsMatch(t, []string{
		"trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T2X",
		"trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T3Y",
	}, read[1].Markers.ReadCodes, "both codes on the page are read")
	assert.Less(t, *read[2].BlankScore, 0.005, "a blank sheet reads as blank")
}
