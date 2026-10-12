package shipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryStatusHasOneStage(t *testing.T) {
	statuses := []Status{
		StatusNew,
		StatusPartiallyAssigned,
		StatusAssigned,
		StatusInTransit,
		StatusDelayed,
		StatusPartiallyCompleted,
		StatusReadyToInvoice,
		StatusCompleted,
		StatusInvoiced,
		StatusCanceled,
	}
	seen := make(map[Status]Stage, len(statuses))
	for _, stage := range Stages() {
		for _, status := range stage.Statuses() {
			_, dup := seen[status]
			require.False(t, dup, "status %s maps to two stages", status)
			seen[status] = stage
		}
	}
	for _, status := range statuses {
		stage, ok := seen[status]
		require.True(t, ok, "status %s has no stage", status)
		assert.Equal(t, stage, StageOf(status))
	}
}

func TestStageRanksAreOrderedAndReversible(t *testing.T) {
	var previous int16
	for _, stage := range Stages() {
		rank := stage.Rank()
		assert.Greater(t, rank, previous)
		back, ok := StageFromRank(rank)
		require.True(t, ok)
		assert.Equal(t, stage, back)
		previous = rank
	}
	_, ok := StageFromRank(99)
	assert.False(t, ok)
}

func TestStageSQLCoversEveryStage(t *testing.T) {
	sql := StageRankSQL("sp.status")
	for _, stage := range Stages() {
		for _, status := range stage.Statuses() {
			assert.Contains(t, sql, "'"+string(status)+"'")
		}
	}
}
