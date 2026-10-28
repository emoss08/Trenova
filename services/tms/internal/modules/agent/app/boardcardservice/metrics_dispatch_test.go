package boardcardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An agent asked for what is late wrote the dispatch board's counts into a
board's prose, which a board refuses, because no figure could read them live.
The dispatch counts are metrics a figures block names, read from the analytics
card the shipment provider writes.
*/
func TestShipmentMetric_ReadsTheDispatchCounts(t *testing.T) {
	t.Parallel()

	cards := &shipmentCards{}
	require.NoError(t, jsonutils.Convert(map[string]any{
		"dispatch": map[string]any{
			"moves": 41, "late": 7, "dueNow": 3, "dueToday": 12, "uncovered": 9,
		},
	}, cards))

	for key, want := range map[string]int64{
		"movesLate": 7, "movesDueNow": 3, "movesDueToday": 12, "movesUncovered": 9,
	} {
		value := shipmentMetric(key, cards)
		require.NotNil(t, value, key)
		assert.Equal(t, board.ValueCount, value.Kind, key)
		assert.Equal(t, want, value.Number.IntPart(), key)

		kind, known := board.MetricKind(key)
		assert.True(t, known, "%s is a metric a board may name", key)
		assert.Equal(t, board.ValueCount, kind, key)
	}
}
