package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnreadableHere_AServerThatCannotReadFilesSkipsTheCase(t *testing.T) {
	t.Parallel()

	for _, code := range []documentupload.FailureCode{
		documentupload.FailureConverterUnavailable,
		documentupload.FailureExtractionUnavailable,
	} {
		err := unreadableHere(&documentcontent.Content{
			Status:         documentcontent.StatusIndexed,
			FailureCode:    string(code),
			FailureMessage: "the document converter is not configured.",
		})
		require.ErrorIs(t, err, ErrAttachmentUnreadable, code)
		assert.Contains(t, err.Error(), "the document converter is not configured.")
	}
}

func TestUnreadableHere_AFileTheServerReadOrRefusedOnItsOwnRuns(t *testing.T) {
	t.Parallel()

	assert.NoError(t, unreadableHere(&documentcontent.Content{
		Status: documentcontent.StatusIndexed,
	}))
	assert.NoError(t, unreadableHere(&documentcontent.Content{
		Status:      documentcontent.StatusFailed,
		FailureCode: string(documentupload.FailureUnsupportedFormat),
	}), "a file the server cannot read is the agent's to say so about")
}
