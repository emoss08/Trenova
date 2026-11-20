package invoicepacket

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/heavyslotstest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewSlot_AViewerWaitsAndAWarmSkipsWhenTheServerIsBusy(t *testing.T) {
	t.Parallel()

	gate := &heavyslotstest.Gate{Busy: true}
	service := &PreviewService{slots: gate}
	warming := context.WithValue(t.Context(), warmingKey{}, true)

	_, err := service.previewSlot(warming)
	require.ErrorIs(t, err, errPreviewBusy, "warming ahead leaves the pages for the viewer")

	_, err = service.previewSlot(t.Context())
	assert.True(t, errortypes.IsRateLimitError(err), "a viewer is told the server is busy")

	gate.Busy = false
	release, err := service.previewSlot(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 1, gate.Held.Load())
	release()
	assert.EqualValues(t, 0, gate.Held.Load())
}

func TestViewerRetries_LoadsAgainForAViewerThatJoinedAGivenUpWarm(t *testing.T) {
	t.Parallel()

	calls := 0
	load := func() (int, error) {
		calls++
		if calls == 1 {
			return 0, errPreviewBusy
		}
		return 7, nil
	}

	got, err := viewerRetries(t.Context(), load)
	require.NoError(t, err)
	assert.Equal(t, 7, got)

	calls = 0
	warming := context.WithValue(t.Context(), warmingKey{}, true)
	_, err = viewerRetries(warming, load)
	require.ErrorIs(t, err, errPreviewBusy)
	assert.Equal(t, 1, calls)
}
