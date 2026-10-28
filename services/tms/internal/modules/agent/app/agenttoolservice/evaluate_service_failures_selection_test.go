package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A model checking one shipment sent it as shipmentId and again as shipmentIds
of that one shipment, and the check was refused for naming both. The two say
the same thing, so the call is read and filed as the one shipment.
*/
func TestEvaluateServiceFailures_ReadsAShipmentNamedTwiceAsOne(t *testing.T) {
	t.Parallel()

	failures := &agenttoolstest.FakeFailureDecider{}
	tool := operationstools.NewEvaluateServiceFailuresTool(failures).(*operationstools.EvaluateServiceFailuresTool)
	shipmentID := pulid.MustNew("shp_").String()

	resolved, err := tool.ResolveSelection(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":  shipmentID,
		"shipmentIds": []any{shipmentID, shipmentID},
		"force":       true,
	}))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"shipmentId": shipmentID, "force": true}, resolved)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":  shipmentID,
		"shipmentIds": []any{shipmentID},
	})))
	require.NotNil(t, failures.Evaluated)
	assert.Equal(t, shipmentID, failures.Evaluated.ShipmentID.String())
	assert.Nil(t, failures.Bulk, "one shipment is not a bulk check")
}

func TestEvaluateServiceFailures_ReadsAListOfOneWithAStopAsThatShipment(t *testing.T) {
	t.Parallel()

	failures := &agenttoolstest.FakeFailureDecider{}
	tool := operationstools.NewEvaluateServiceFailuresTool(failures).(*operationstools.EvaluateServiceFailuresTool)
	shipmentID, stopID := pulid.MustNew("shp_"), pulid.MustNew("stp_")
	params := map[string]any{
		"shipmentIds":          []any{shipmentID.String()},
		toolparams.ParamStopID: stopID.String(),
	}

	resolved, err := tool.ResolveSelection(t.Context(), agenttoolstest.ExecuteParams(params))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"shipmentId":           shipmentID.String(),
		toolparams.ParamStopID: stopID.String(),
	}, resolved)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(params)))
	require.NotNil(t, failures.EvaluatedStop)
	assert.Equal(t, shipmentID, failures.EvaluatedStop.ShipmentID)
	assert.Equal(t, stopID, failures.EvaluatedStop.StopID)
}

func TestEvaluateServiceFailures_LeavesASeveralShipmentCheckAsItIs(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewEvaluateServiceFailuresTool(&agenttoolstest.FakeFailureDecider{}).(*operationstools.EvaluateServiceFailuresTool)
	params := map[string]any{"shipmentIds": agenttoolstest.ManyIDs(2)}

	resolved, err := tool.ResolveSelection(t.Context(), agenttoolstest.ExecuteParams(params))
	require.NoError(t, err)
	assert.Equal(t, params, resolved)
}

func TestEvaluateServiceFailures_ARefusalForBothScopesShowsBothCalls(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewEvaluateServiceFailuresTool(&agenttoolstest.FakeFailureDecider{}).(*operationstools.EvaluateServiceFailuresTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":  pulid.MustNew("shp_").String(),
		"shipmentIds": agenttoolstest.ManyIDs(2),
	})

	resolved, err := tool.ResolveSelection(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, params.Params, resolved, "Validate is left to refuse it")

	err = tool.Validate(t.Context(), params)
	require.ErrorIs(t, err, operationstools.ErrBothEvaluationScopes)
	assert.Contains(t, err.Error(), "send shipmentId without shipmentIds")
	assert.Contains(t, err.Error(), "send shipmentIds without shipmentId")
}
