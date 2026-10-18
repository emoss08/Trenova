package deskbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweepTools_KeepsWhatTheTaskNeedsAndFillsWithTheRest(t *testing.T) {
	t.Parallel()

	base := []string{
		"list_shipments", "get_shipment", "search_shipments", "assign_move", "list_workers",
		"get_worker_hos", "add_shipment_comment", "recall_memory", "list_trailers",
	}

	tools, err := sweepTools(base, []string{"get_shipment|search_shipments", "add_shipment_comment",
		"recall_memory"}, 4)
	require.NoError(t, err)
	assert.Len(t, tools, 4)
	assert.Contains(t, tools, "get_shipment")
	assert.Contains(t, tools, "add_shipment_comment")
	assert.NotContains(t, tools, "recall_memory", "core tools ride on every turn anyway")
	assert.IsIncreasing(t, tools)

	_, err = sweepTools(base, []string{"get_shipment", "assign_move", "list_workers"}, 2)
	require.ErrorIs(t, err, ErrSweepSkipped)

	_, err = sweepTools(base, nil, 30)
	require.ErrorIs(t, err, ErrSweepSkipped, "the agent holds fewer tools than the cell")
}

func TestSweepMode_Threshold(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AlwaysDisclose, SweepDisclosed.Threshold())
	assert.Equal(t, NeverDisclose, SweepAll.Threshold())
}
