package benchscenario

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweepTools_KeepsWhatTheTaskNeedsAndFillsWithTheRest(t *testing.T) {
	t.Parallel()

	base := []string{
		"shipment_list", "shipment_get", "shipment_get_tracking", "dispatch_assign_move", "worker_list",
		"worker_get_hos", "shipment_add_comment", "agent_list_memories", "equipment_list_trailers",
	}

	tools, err := SweepTools(base, []string{"shipment_get|shipment_list", "shipment_add_comment",
		"agent_list_memories"}, 4)
	require.NoError(t, err)
	assert.Len(t, tools, 4)
	assert.Contains(t, tools, "shipment_get")
	assert.Contains(t, tools, "shipment_add_comment")
	assert.NotContains(t, tools, "agent_list_memories", "core tools ride on every turn anyway")
	assert.IsIncreasing(t, tools)

	_, err = SweepTools(base, []string{"shipment_get", "dispatch_assign_move", "worker_list"}, 2)
	require.ErrorIs(t, err, ErrSweepSkipped)

	_, err = SweepTools(base, nil, 30)
	require.ErrorIs(t, err, ErrSweepSkipped, "the agent holds fewer tools than the cell")
}

func TestSweepMode_Threshold(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AlwaysDisclose, SweepDisclosed.Threshold())
	assert.Equal(t, NeverDisclose, SweepAll.Threshold())
}
