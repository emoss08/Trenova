package proposalexecutor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reportingTool is a write that names what it made.
type reportingTool struct {
	recordingTool

	result *agent.ToolExecutionResult
}

func (t *reportingTool) ExecuteWithResult(
	ctx context.Context,
	params services.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	if err := t.Execute(ctx, params); err != nil {
		return nil, err
	}

	return t.result, nil
}

func newReportingTool(result *agent.ToolExecutionResult, err error) *reportingTool {
	return &reportingTool{
		recordingTool: recordingTool{
			name:      "create_report",
			resource:  permission.ResourceReport,
			operation: permission.OpCreate,
			err:       err,
		},
		result: result,
	}
}

/*
An approved write that names what it made has that recorded with it.

create_report was approved and ran, and all the conversation learned was that
it ran. The agent then passed the proposal's id to describe_report in place of
the report's. The result is written in the same update that marks the proposal
executed, so nothing can read it as executed without it.
*/
func TestExecute_RecordsWhatTheToolMade(t *testing.T) {
	t.Parallel()

	tool := newReportingTool(&agent.ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   "Shipments for\nPeak Distributing",
		IDs:    map[string]string{"definitionId": "rd_01"},
	}, nil)
	repo := &fakeProposalRepo{}
	proposal := testProposal("create_report", map[string]any{"name": "Shipments"})

	err := newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, tool.calls)
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status)
	require.NotNil(t, repo.recorded[0].result)
	assert.Equal(t, "Shipments for Peak Distributing", repo.recorded[0].result.Name,
		"what is recorded is bounded to one line")
	assert.Equal(t, map[string]string{"definitionId": "rd_01"}, repo.recorded[0].result.IDs)
}

// However much a tool reports, a proposal keeps a pointer to the record and
// not the record.
func TestExecute_BoundsWhatTheToolReports(t *testing.T) {
	t.Parallel()

	ids := make(map[string]string, 12)
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		ids[key+"Id"] = strings.Repeat("x", 500)
	}
	tool := newReportingTool(&agent.ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   strings.Repeat("n", 5000),
		IDs:    ids,
	}, nil)
	repo := &fakeProposalRepo{}
	proposal := testProposal("create_report", map[string]any{})

	require.NoError(t, newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	))

	require.Len(t, repo.recorded, 1)
	result := repo.recorded[0].result
	require.NotNil(t, result)
	assert.LessOrEqual(t, len([]rune(result.Name)), 200)
	assert.LessOrEqual(t, len(result.IDs), 8)
	size := len(result.Action) + len(result.Kind) + len(result.Name)
	for key, id := range result.IDs {
		size += len(key) + len(id)
	}
	assert.LessOrEqual(t, size, 2048)
}

// A write that failed made nothing, and the failure clears whatever an
// earlier run of the same proposal reported.
func TestExecute_RecordsNoResultWhenTheToolFails(t *testing.T) {
	t.Parallel()

	tool := newReportingTool(
		&agent.ToolExecutionResult{Action: "created", Kind: "report"},
		errors.New("name taken"),
	)
	repo := &fakeProposalRepo{}
	proposal := testProposal("create_report", map[string]any{})

	err := newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.Error(t, err)

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
	assert.Nil(t, repo.recorded[0].result)
}

// A tool that names nothing still runs, and records no result.
func TestExecute_RecordsNoResultForAToolThatReportsNone(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	proposal := testProposal("reassign_move", map[string]any{})

	require.NoError(t, newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	))

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status)
	assert.Nil(t, repo.recorded[0].result)
}
