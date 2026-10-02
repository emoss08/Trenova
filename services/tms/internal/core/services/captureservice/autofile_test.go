package captureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAutoFileable_ScanIntoAConversationFilesEveryItem(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	requestID := pulid.MustNew("creq_")
	batch := &capture.CaptureBatch{RequestID: &requestID}
	items := []*capture.CaptureItem{
		{
			ID:               pulid.MustNew("capi_"),
			SuggestedType:    capture.ResourceAssistantThread,
			SuggestedID:      &threadID,
			SuggestionSource: capture.SuggestionRequest,
		},
		{
			ID:               pulid.MustNew("capi_"),
			SuggestedType:    capture.ResourceAssistantThread,
			SuggestedID:      &threadID,
			SuggestionSource: capture.SuggestionRequest,
		},
	}

	assert.Equal(t, []pulid.ID{items[0].ID, items[1].ID}, autoFileable(batch, items, false))
}

func TestAutoFileable_ConversationScanWithoutARequestWaits(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	items := []*capture.CaptureItem{{
		ID:               pulid.MustNew("capi_"),
		SuggestedType:    capture.ResourceAssistantThread,
		SuggestedID:      &threadID,
		SuggestionSource: capture.SuggestionClassifier,
	}}

	assert.Empty(t, autoFileable(&capture.CaptureBatch{}, items, true))
}
