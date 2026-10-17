package watchtower

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handedItem(t *testing.T) (*Item, *agent.AgentRun) {
	t.Helper()

	item := openItem()
	run := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	require.NoError(t, item.HandTo(pulid.MustNew("adef_"), run.ID, pulid.MustNew("usr_"), testNow))

	return item, run
}

func TestAgentWorkFrom_FollowsTheRunThroughItsPhases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status agent.RunStatus
		want   AgentStatus
		step   int
		failed bool
	}{
		{agent.RunStatusPending, AgentStatusWorking, 0, false},
		{agent.RunStatusGatheringContext, AgentStatusWorking, 0, false},
		{agent.RunStatusDiagnosing, AgentStatusWorking, 1, false},
		{agent.RunStatusAwaitingDecision, AgentStatusNeedsYou, 2, false},
		{agent.RunStatusFailed, AgentStatusNeedsYou, 1, true},
		{agent.RunStatusCompleted, AgentStatusDone, 3, false},
		{agent.RunStatusShadowCompleted, AgentStatusDone, 3, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			item, run := handedItem(t)
			run.Status = tc.status
			work := item.AgentWorkFrom(run)

			require.NotNil(t, work)
			assert.Equal(t, tc.want, work.Status)
			assert.Equal(t, tc.step, work.Step)
			assert.Equal(t, tc.failed, work.Failed)
			assert.Equal(
				t,
				[]AgentStep{AgentStepRead, AgentStepDecide, AgentStepApprove},
				work.Steps,
			)
			assert.Equal(t, item.AgentRunID, work.RunID)
			assert.Equal(t, testNow, work.HandedAt)
		})
	}
}

func TestAgentWorkFrom_SaysWhyARunStoppedAndWhatAFinishedOneSaid(t *testing.T) {
	t.Parallel()

	item, run := handedItem(t)
	run.Status = agent.RunStatusFailed
	run.ErrorMessage = "Bank feed token expired\nstack trace follows"
	assert.Equal(t, "Bank feed token expired", item.AgentWorkFrom(run).Note)

	run.ErrorMessage = ""
	run.Summary = "  No driver is free.\nBoth moves are overdue.  "
	assert.Equal(t, "No driver is free.\nBoth moves are overdue.", item.AgentWorkFrom(run).Note)

	run.Status = agent.RunStatusCompleted
	run.Summary = strings.Repeat("a", 400) + "\n\n" + strings.Repeat("b", 400)
	assert.Equal(t, run.Summary, item.AgentWorkFrom(run).Note)
}

func TestAgentWorkFrom_ARunNoLongerKeptReadsAsDone(t *testing.T) {
	t.Parallel()

	item, _ := handedItem(t)
	work := item.AgentWorkFrom(nil)

	assert.Equal(t, AgentStatusDone, work.Status)
	assert.Equal(t, len(work.Steps), work.Step)
	assert.False(t, work.Active())
}

func TestAgentWorkFrom_NothingWithoutAnAgent(t *testing.T) {
	t.Parallel()

	assert.Nil(t, openItem().AgentWorkFrom(&agent.AgentRun{}))
}

func TestHandTo_StartsTheExceptionAndRefusesAResolvedOne(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.NoError(t, item.Snooze(testNow+600, testNow))
	require.NoError(
		t,
		item.HandTo(pulid.MustNew("adef_"), pulid.MustNew("ar_"), pulid.Nil, testNow),
	)
	assert.Equal(t, StatusInProgress, item.Status)
	assert.Nil(t, item.SnoozedUntil)
	require.NotNil(t, item.AcknowledgedAt)

	resolved := openItem()
	require.NoError(t, resolved.Resolve(ResolutionFixed, "", pulid.Nil, testNow))
	require.Error(
		t,
		resolved.HandTo(pulid.MustNew("adef_"), pulid.MustNew("ar_"), pulid.Nil, testNow),
	)
}
