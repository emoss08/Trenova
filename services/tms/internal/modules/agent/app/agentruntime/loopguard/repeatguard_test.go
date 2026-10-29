package loopguard

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Key order is a serialisation detail of whichever provider produced the call,
// not a difference in what was asked.
func TestCallKey_IgnoresArgumentOrder(t *testing.T) {
	t.Parallel()

	first, ok := CallKey(&llmports.ToolCall{
		Name:      "report_run",
		Arguments: map[string]any{"a": 1, "b": 2},
	})
	require.True(t, ok)
	second, ok := CallKey(&llmports.ToolCall{
		Name:      "report_run",
		Arguments: map[string]any{"b": 2, "a": 1},
	})
	require.True(t, ok)

	assert.Equal(t, first, second)
}

func TestCallKey_SeparatesToolsThatShareArguments(t *testing.T) {
	t.Parallel()

	args := map[string]any{"id": "shp_1"}
	first, _ := CallKey(&llmports.ToolCall{Name: "shipment_get", Arguments: args})
	second, _ := CallKey(&llmports.ToolCall{Name: "shipment_cancel", Arguments: args})

	assert.NotEqual(t, first, second)
}

func TestFanOutNote_RemindsOnlyFromTheThirdSingleFetch(t *testing.T) {
	t.Parallel()

	assert.Empty(t, FanOutNote("billing_get_queue_item", 2))
	assert.Empty(t, FanOutNote("billing_list_queue_items", 5))
	assert.Contains(t, FanOutNote("billing_get_queue_item", 3), "3rd billing_get_queue_item call")
	assert.Contains(t, FanOutNote("billing_get_queue_item", 12), "12th")
}

func TestRepeatGuard_AnswersARepeatedReadUntilSomethingWrites(t *testing.T) {
	t.Parallel()

	guard := NewRepeatGuard()
	read := llmports.ToolCall{
		Name:      "billing_get_queue_item",
		Arguments: map[string]any{"billingQueueItemId": "bqi_1"},
	}
	other := llmports.ToolCall{
		Name:      "billing_get_queue_item",
		Arguments: map[string]any{"billingQueueItemId": "bqi_2"},
	}
	write := llmports.ToolCall{
		Name:      "billing_approve_queue_item",
		Arguments: map[string]any{"billingQueueItemId": "bqi_1"},
	}

	assert.False(t, guard.ReadBefore(&read), "a first read runs")
	guard.Ran(&read)
	assert.True(t, guard.ReadBefore(&read), "the same read again is answered from the first")
	assert.False(t, guard.ReadBefore(&other), "a read with other arguments runs")

	guard.Ran(&write)
	assert.False(
		t,
		guard.ReadBefore(&read),
		"after a write the record may have changed, so it is read again",
	)
	assert.False(t, guard.ReadBefore(&write), "a write is never answered from an earlier one")
}
