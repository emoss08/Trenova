package proposalexecutor

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// targetedSchemaTool is schemaTool naming the driver it messages as its
// target, the way a tool that acts on one record does.
type targetedSchemaTool struct {
	schemaTool
}

func (t *targetedSchemaTool) Target(params map[string]any) (services.ToolTarget, bool) {
	raw, _ := params["workerId"].(string)
	id, err := pulid.Parse(raw)
	if err != nil {
		return services.ToolTarget{}, false
	}

	return services.ToolTarget{Resource: permission.ResourceWorker, ID: id}, true
}

func retargetFixture() (*targetedSchemaTool, *Service, pulid.ID) {
	tool := &targetedSchemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})

	return tool, executor, pulid.MustNew("wrk_")
}

// Approving with changes used to accept a new driver id: the write went to a
// driver nobody proposed a message for, while the staleness check still
// compared the driver that was pinned.
func TestCheckModifications_RefusesPointingTheChangeAtAnotherRecord(t *testing.T) {
	t.Parallel()

	tool, executor, driver := retargetFixture()
	proposal := testProposal(tool.Name(), map[string]any{
		"workerId": driver.String(),
		"message":  "Call in",
	})

	_, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"workerId": pulid.MustNew("wrk_").String()},
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "workerId", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrForbidden, multiErr.Errors[0].Code)
}

func TestExecute_RefusesARetargetedApproval(t *testing.T) {
	t.Parallel()

	tool, executor, driver := retargetFixture()
	proposal := testProposal(tool.Name(), map[string]any{
		"workerId": driver.String(),
		"message":  "Call in",
	})

	err := executor.Execute(
		t.Context(),
		proposal,
		map[string]any{"workerId": pulid.MustNew("wrk_").String()},
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, tool.ran, "the write never reaches another record")
}

func TestCheckModifications_AcceptsChangesThatKeepTheRecord(t *testing.T) {
	t.Parallel()

	tool, executor, driver := retargetFixture()
	proposal := testProposal(tool.Name(), map[string]any{
		"workerId": driver.String(),
		"message":  "Call in",
	})

	params, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"message": "Call dispatch", "workerId": driver.String()},
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, "Call dispatch", params["message"])
}

func TestProposalFields_MakesTheTargetReadOnly(t *testing.T) {
	t.Parallel()

	tool, _, driver := retargetFixture()

	fields := services.ProposalFields(tool, map[string]any{
		"workerId": driver.String(),
		"message":  "Call in",
	})

	readOnly := map[string]bool{}
	for _, field := range fields {
		readOnly[field.Name] = field.ReadOnly
	}
	assert.True(t, readOnly["workerId"])
	assert.False(t, readOnly["message"])
	assert.False(t, readOnly["priority"])
}

// A plan's later step on a record an earlier step changed runs against the
// version that step left, not the one pinned before either ran.
func TestRun_HonoursTheVersionAnEarlierStepLeft(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "place_shipment_hold",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.versions = &fakeVersions{version: 4}

	proposal := pinnedProposal("place_shipment_hold", 3)
	expected := int64(4)
	outcome, err := executor.Run(t.Context(), &Approval{
		Proposal:              proposal,
		Actor:                 testActor(proposal.OrganizationID, proposal.BusinessUnitID),
		ExpectedTargetVersion: &expected,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, tool.calls)
	require.NotNil(t, outcome.TargetVersion)
	assert.Equal(t, int64(4), *outcome.TargetVersion)

	stale := int64(2)
	again := pinnedProposal("place_shipment_hold", 3)
	_, err = executor.Run(t.Context(), &Approval{
		Proposal:              again,
		Actor:                 testActor(again.OrganizationID, again.BusinessUnitID),
		ExpectedTargetVersion: &stale,
	})
	require.ErrorIs(t, err, ErrTargetChanged)
}
