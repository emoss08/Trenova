package watchtowersources

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingFailedRuns struct {
	agentports.AgentRunRepository

	requests []*agentports.ListAgentRunRequest
}

func (r *recordingFailedRuns) List(
	_ context.Context,
	req *agentports.ListAgentRunRequest,
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

	require.Len(t, runs.requests, 1)
	assert.Equal(t, agentports.PersonRunRead(pulid.Nil), runs.requests[0].Reader,
		"a failed run's summary can repeat a private conversation, and the tower is "+
			"read by everyone who may read agent runs")
}
