package agentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activitySummary() *services.AgentActivitySummary {
	oldest := int64(1_799_996_400)
	share := 0.75

	return &services.AgentActivitySummary{
		Since:              1_799_967_600,
		Runs:               14,
		RunsFailed:         2,
		RunsWorking:        1,
		RunsAwaiting:       3,
		PendingProposals:   4,
		OldestPendingAt:    &oldest,
		OpenExceptions:     2,
		DecisionWindowDays: 7,
		Decided:            8,
		ApprovedAsProposed: &share,
	}
}

func TestActivitySummaryShowsWhatTheReaderMaySee(t *testing.T) {
	t.Parallel()

	out := ActivitySummaryToModel(activitySummary(), ActivityVisibility{Proposals: true, Exceptions: true})

	assert.Equal(t, 14, out.Runs)
	assert.Equal(t, 2, out.RunsFailed)
	require.NotNil(t, out.PendingProposals)
	assert.Equal(t, 4, *out.PendingProposals)
	require.NotNil(t, out.OldestPendingAt)
	assert.Equal(t, 1_799_996_400, *out.OldestPendingAt)
	require.NotNil(t, out.OpenExceptions)
	assert.Equal(t, 2, *out.OpenExceptions)
	require.NotNil(t, out.ApprovedAsProposed)
	assert.InDelta(t, 0.75, *out.ApprovedAsProposed, 1e-9)
}

func TestActivitySummaryLeavesOutProposalsAndExceptionsTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	out := ActivitySummaryToModel(activitySummary(), ActivityVisibility{})

	assert.Equal(t, 14, out.Runs)
	assert.Nil(t, out.PendingProposals)
	assert.Nil(t, out.OldestPendingAt)
	assert.Nil(t, out.Decided)
	assert.Nil(t, out.ApprovedAsProposed)
	assert.Nil(t, out.OpenExceptions)
}
