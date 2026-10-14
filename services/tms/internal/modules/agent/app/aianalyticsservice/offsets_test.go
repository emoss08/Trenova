package aianalyticsservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOffsetSpansFollowADaylightSavingChangeInsideTheWindow(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	since := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC).Unix()
	until := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC).Unix()

	spans := offsetSpans(chicago, since, until)

	require.Len(t, spans, 2)
	assert.Equal(t, aianalyticsports.OffsetSpan{From: since, Offset: -5 * 3600}, spans[0])
	assert.Equal(t, time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC).Unix(), spans[1].From)
	assert.Equal(t, int64(-6*3600), spans[1].Offset)
}

func TestOffsetSpansInUTCAreOneSpanOfNothing(t *testing.T) {
	t.Parallel()

	spans := offsetSpans(time.UTC, 1_790_000_000, 1_792_592_000)

	assert.Equal(t, []aianalyticsports.OffsetSpan{{From: 1_790_000_000, Offset: 0}}, spans)
}
