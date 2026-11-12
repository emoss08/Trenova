package watchtowersources

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingFailedRuns struct {
	agentruntimeports.AgentRunRepository

	requests []*agentruntimeports.ListAgentRunRequest
}

func (r *recordingFailedRuns) List(
	_ context.Context,
	req *agentruntimeports.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	r.requests = append(r.requests, req)

	return &pagination.ListResult[*agent.AgentRun]{}, nil
}

func TestFailedRunSource_LeavesConversationRunsOffTheSharedTower(t *testing.T) {
	t.Parallel()

	runs := &recordingFailedRuns{}
	source := &FailedRunSource{runs: runs, now: func() int64 { return 1_000_000 }}

	_, err := source.Snapshot(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	require.NotEmpty(t, runs.requests)
	assert.Equal(t, agentruntimeports.PersonRunRead(pulid.Nil), runs.requests[0].Reader,
		"a failed run's summary can repeat a private conversation, and the tower is "+
			"read by everyone who may read agent runs")
}

func TestFailuresNotGotPast_AFailureIsNewsUntilTheAgentFinishesARunAfterIt(t *testing.T) {
	t.Parallel()

	billing, dispatch, monitor := pulid.MustNew("adef_"), pulid.MustNew("adef_"), pulid.MustNew("adef_")
	run := func(definition pulid.ID, at int64) *agent.AgentRun {
		return &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: definition, CreatedAt: at}
	}
	recovered := run(billing, 100)
	stillDown := run(dispatch, 200)
	failedLast := run(monitor, 300)

	kept := failuresNotGotPast(
		[]*agent.AgentRun{recovered, stillDown, failedLast},
		[]*agent.AgentRun{run(billing, 150), run(billing, 400), run(monitor, 250), nil},
	)

	assert.Equal(t, []*agent.AgentRun{stillDown, failedLast}, kept,
		"billing finished a run after it failed; dispatch never has; monitor's last good run "+
			"was before its failure")
}

func TestFailedRunSource_AsksOnlyAboutTheAgentsThatFailed(t *testing.T) {
	t.Parallel()

	definition := pulid.MustNew("adef_")
	failed := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: definition, CreatedAt: 900_000}
	runs := &scriptedRuns{pages: [][]*agent.AgentRun{{failed}, {}}}
	source := &FailedRunSource{runs: runs, now: func() int64 { return 1_000_000 }}

	items, err := source.Snapshot(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)

	require.Len(t, items, 1)
	require.Len(t, runs.requests, 2)
	second := runs.requests[1].Filter.FieldFilters
	assert.Equal(t, []string{definition.String()}, second[1].Value)
	assert.Equal(t, int64(900_000), second[2].Value)
}

type scriptedRuns struct {
	agentruntimeports.AgentRunRepository

	pages    [][]*agent.AgentRun
	requests []*agentruntimeports.ListAgentRunRequest
}

func (r *scriptedRuns) List(
	_ context.Context,
	req *agentruntimeports.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	r.requests = append(r.requests, req)
	page := r.pages[len(r.requests)-1]

	return &pagination.ListResult[*agent.AgentRun]{Items: page}, nil
}
