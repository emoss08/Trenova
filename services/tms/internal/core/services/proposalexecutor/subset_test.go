package proposalexecutor

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// subsetTool transfers a set of shipments, the way transfer_to_billing does:
// its ids are a subset field an approver may narrow before approving.
type subsetTool struct {
	ran map[string]any
}

func (t *subsetTool) Name() string        { return "transfer_shipments" }
func (t *subsetTool) Description() string { return "Transfer shipments." }
func (t *subsetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentIds": toolschema.RecordSubset(permission.ResourceShipment.String(),
				map[string]any{
					"type":     "array",
					"minItems": 1,
					"items":    map[string]any{"type": "string"},
				}),
		},
		"required":             []string{"shipmentIds"},
		"additionalProperties": false,
	}
}
func (t *subsetTool) Policy() services.ToolPolicy {
	return services.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "A subset stub.",
	}
}
func (t *subsetTool) Execute(_ context.Context, params services.ToolExecuteParams) error {
	t.ran = params.Params

	return nil
}

func subsetFixture() (*subsetTool, *Service, *agent.AgentProposal, *services.RequestActor) {
	tool := &subsetTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{
		"shipmentIds": []any{"shp_a", "shp_b", "shp_c"},
	})

	return tool, executor, proposal, testActor(proposal.OrganizationID, proposal.BusinessUnitID)
}

// A person approving a transfer of three shipments may untick one: what runs
// is still inside what the agent proposed.
func TestCheckModifications_AcceptsNarrowingARecordSubset(t *testing.T) {
	t.Parallel()

	_, executor, proposal, actor := subsetFixture()

	params, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"shipmentIds": []any{"shp_a", "shp_c"}},
		actor,
	)

	require.NoError(t, err)
	assert.Equal(t, []any{"shp_a", "shp_c"}, params["shipmentIds"])
}

// Adding a shipment would approve a transfer nobody proposed.
func TestCheckModifications_RefusesWideningARecordSubset(t *testing.T) {
	t.Parallel()

	_, executor, proposal, actor := subsetFixture()

	_, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"shipmentIds": []any{"shp_a", "shp_other"}},
		actor,
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "shipmentIds", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrForbidden, multiErr.Errors[0].Code)
}

func TestCheckModifications_RefusesEmptyingARecordSubset(t *testing.T) {
	t.Parallel()

	_, executor, proposal, actor := subsetFixture()

	_, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"shipmentIds": []any{}},
		actor,
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "shipmentIds", multiErr.Errors[0].Field)
}

// The rule holds where the write runs too, whatever the decision carried.
func TestExecute_RefusesAWidenedRecordSubset(t *testing.T) {
	t.Parallel()

	tool, executor, proposal, actor := subsetFixture()

	err := executor.Execute(
		t.Context(),
		proposal,
		map[string]any{"shipmentIds": []any{"shp_a", "shp_other"}},
		actor,
	)

	require.Error(t, err)
	assert.Nil(t, tool.ran, "no shipment outside the proposal is transferred")
}

func TestExecute_RunsANarrowedRecordSubset(t *testing.T) {
	t.Parallel()

	tool, executor, proposal, actor := subsetFixture()

	require.NoError(t, executor.Execute(
		t.Context(),
		proposal,
		map[string]any{"shipmentIds": []any{"shp_b"}},
		actor,
	))
	assert.Equal(t, []any{"shp_b"}, tool.ran["shipmentIds"])
}
