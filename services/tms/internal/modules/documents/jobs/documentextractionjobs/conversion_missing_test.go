package documentextractionjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A document whose file is gone from storage cannot be read on any attempt. The
conversion records that on the document the first time it looks, without
failing the activity, so no retry is spent waiting for a file that will not
come back and the converter is never called.
*/
func TestProcessDocumentExtractionWorkflow_AMissingFileFailsTheDocumentOnTheFirstAttempt(
	t *testing.T,
) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	h.converter.EXPECT().Enabled().Return(true).Maybe()
	imaging := imagingfake.New()
	h.activities.imaging = imaging
	h.activities.conversionCfg = &documentconversionconfig.Config{
		DoclingURL: "http://docling:5001",
		MaxPages:   10,
	}
	require.NoError(t, h.storage.Delete(t.Context(), h.docs.Doc.StoragePath))

	result := h.run(t)

	require.NotNil(t, result)
	assert.Equal(t, string(documentcontent.StatusFailed), result.Status)
	require.NotNil(t, h.contents.Content)
	assert.Equal(t, documentcontent.StatusFailed, h.contents.Content.Status)
	assert.Equal(
		t,
		documentupload.FailureDownloadFailed.String(),
		h.contents.Content.FailureCode,
	)
	assert.Equal(t, missingFileMessage, h.contents.Content.FailureMessage)
	assert.Empty(t, h.contents.Pages)
	assert.Equal(t, 1, imaging.CallCount("probe"), "the conversion ran once")
}
