package agentwork_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwork"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/shared/pulid"
)

const testNow = int64(1_700_000_000)

func openItem() *watchtower.Item {
	return &watchtower.Item{
		ID:             pulid.MustNew("wt_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		SourceKind:     watchtower.SourceServiceFailure,
		SourceID:       "sf_1",
		Severity:       watchtower.SeverityWarning,
		Status:         watchtower.StatusOpen,
	}
}

func handedItem(t *testing.T) (*watchtower.Item, *agent.AgentRun) {
	t.Helper()

	item := openItem()
	run := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	require.NoError(t, item.HandTo(pulid.MustNew("adef_"), run.ID, pulid.MustNew("usr_"), testNow))

	return item, run
}

func TestFrom_FollowsTheRunThroughItsPhases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status agent.RunStatus
		want   agentwork.Status
		step   int
		failed bool
	}{
		{agent.RunStatusPending, agentwork.StatusWorking, 0, false},
		{agent.RunStatusGatheringContext, agentwork.StatusWorking, 0, false},
		{agent.RunStatusDiagnosing, agentwork.StatusWorking, 1, false},
		{agent.RunStatusAwaitingDecision, agentwork.StatusNeedsYou, 2, false},
		{agent.RunStatusFailed, agentwork.StatusNeedsYou, 1, true},
		{agent.RunStatusCompleted, agentwork.StatusDone, 3, false},
		{agent.RunStatusShadowCompleted, agentwork.StatusDone, 3, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			item, run := handedItem(t)
			run.Status = tc.status
			work := agentwork.From(item, run)

			require.NotNil(t, work)
			assert.Equal(t, tc.want, work.Status)
			assert.Equal(t, tc.step, work.Step)
			assert.Equal(t, tc.failed, work.Failed)
			assert.Equal(
				t,
				[]agentwork.Step{agentwork.StepRead, agentwork.StepDecide, agentwork.StepApprove},
				work.Steps,
			)
			assert.Equal(t, item.AgentRunID, work.RunID)
			assert.Equal(t, testNow, work.HandedAt)
		})
	}
}

func TestFrom_SaysWhyARunStoppedAndWhatAFinishedOneSaid(t *testing.T) {
	t.Parallel()

	item, run := handedItem(t)
	run.Status = agent.RunStatusFailed
	run.ErrorMessage = "Bank feed token expired\nstack trace follows"
	assert.Equal(t, "Bank feed token expired", agentwork.From(item, run).Note)

	run.ErrorMessage = ""
	run.Summary = "  No driver is free.\nBoth moves are overdue.  "
	assert.Equal(t, "No driver is free.\nBoth moves are overdue.", agentwork.From(item, run).Note)

	run.Status = agent.RunStatusCompleted
	run.Summary = strings.Repeat("a", 400) + "\n\n" + strings.Repeat("b", 400)
	assert.Equal(t, run.Summary, agentwork.From(item, run).Note)
}

func TestFrom_ARunNoLongerKeptReadsAsDone(t *testing.T) {
	t.Parallel()

	item, _ := handedItem(t)
	work := agentwork.From(item, nil)

	assert.Equal(t, agentwork.StatusDone, work.Status)
	assert.Equal(t, len(work.Steps), work.Step)
	assert.False(t, work.Active())
}

func TestFrom_NothingWithoutAnAgent(t *testing.T) {
	t.Parallel()

	assert.Nil(t, agentwork.From(openItem(), &agent.AgentRun{}))
}

func TestFrom_WorksExactlyWhileTheRunIsWorking(t *testing.T) {
	t.Parallel()

	statuses := []agent.RunStatus{
		agent.RunStatusPending,
		agent.RunStatusGatheringContext,
		agent.RunStatusDiagnosing,
		agent.RunStatusAwaitingDecision,
		agent.RunStatusCompleted,
		agent.RunStatusShadowCompleted,
		agent.RunStatusFailed,
		agent.RunStatus("SomethingNew"),
	}
	for _, status := range statuses {
		item, run := handedItem(t)
		run.Status = status

		assert.Equal(t, status.IsWorking(), agentwork.From(item, run).Active(), string(status))
	}
	assert.Equal(
		t,
		[]agent.RunStatus{
			agent.RunStatusPending,
			agent.RunStatusGatheringContext,
			agent.RunStatusDiagnosing,
		},
		agent.WorkingRunStatuses(),
	)
}

// An agent that finished on an item is not run on it again while the item
// says nothing new: its answer would be the same. A failed run answered
// nothing, and an item described again since the hand-off is news.
func TestAnsweredUnchanged_OnlyAFinishedRunOnAnItemThatSaysNothingNew(t *testing.T) {
	t.Parallel()

	item, run := handedItem(t)
	item.DescribedAt = testNow - 60

	run.Status = agent.RunStatusCompleted
	assert.True(t, agentwork.AnsweredUnchanged(item, run))

	run.Status = agent.RunStatusFailed
	assert.False(t, agentwork.AnsweredUnchanged(item, run), "a failed run answered nothing")

	run.Status = agent.RunStatusDiagnosing
	assert.False(t, agentwork.AnsweredUnchanged(item, run), "a run still working has not answered")

	run.Status = agent.RunStatusCompleted
	item.DescribedAt = testNow + 60
	assert.False(t, agentwork.AnsweredUnchanged(item, run), "the item says something new")

	assert.False(t, agentwork.AnsweredUnchanged(item, nil))
	assert.False(t, agentwork.AnsweredUnchanged(item, &agent.AgentRun{ID: pulid.MustNew("ar_")}))
	assert.False(t, agentwork.AnsweredUnchanged(openItem(), run), "nobody was handed it")
}
