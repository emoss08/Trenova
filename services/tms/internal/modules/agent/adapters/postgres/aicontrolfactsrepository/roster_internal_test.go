package aicontrolfactsrepository

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleRosterMergesRunsAndDecisionsPerAgent(t *testing.T) {
	t.Parallel()

	busy := pulid.MustNew("agdef_")
	quiet := pulid.MustNew("agdef_")
	shadow := pulid.MustNew("agdef_")

	stats := assembleRoster(
		[]rosterRunRow{
			{AgentID: busy.String(), Day: 0, Runs: 2},
			{AgentID: busy.String(), Day: 13, Runs: 5},
			{AgentID: quiet.String(), Day: 3, Runs: 1},
			{AgentID: "not-an-id", Day: 1, Runs: 9},
		},
		[]rosterDecisionRow{
			{AgentID: busy.String(), Approved: 18, Modified: 6, Rejected: 1, Failed: 2},
			{AgentID: shadow.String(), Shadow: 31},
		},
	)

	require.Len(t, stats, 3)
	assert.Equal(t, busy, stats[0].AgentID)
	assert.Equal(t, 7, stats[0].Runs())
	assert.Equal(t, 5, stats[0].RunsByDay[13])
	assert.Equal(t, 18, stats[0].Approved)
	assert.Equal(t, 2, stats[0].Failed)
	assert.Equal(t, quiet, stats[1].AgentID)
	assert.Equal(t, 1, stats[1].RunsByDay[3])
	assert.Nil(t, stats[1].ApprovalRate())
	assert.Equal(t, shadow, stats[2].AgentID)
	assert.Equal(t, 31, stats[2].ShadowRecorded)
	assert.Equal(t, 0, stats[2].Runs())
}
