package agenttoolservice

import (
	"testing"

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

	failures := &fakeFailureDecider{}
	tool := newEvaluateServiceFailuresTool(failures).(*evaluateServiceFailuresTool)
	shipmentID := pulid.MustNew("shp_").String()

	resolved, err := tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		"shipmentId":  shipmentID,
		"shipmentIds": []any{shipmentID, shipmentID},
		"force":       true,
	}))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"shipmentId": shipmentID, "force": true}, resolved)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId":  shipmentID,
		"shipmentIds": []any{shipmentID},
	})))
	require.NotNil(t, failures.evaluated)
	assert.Equal(t, shipmentID, failures.evaluated.ShipmentID.String())
	assert.Nil(t, failures.bulk, "one shipment is not a bulk check")
}

func TestEvaluateServiceFailures_ReadsAListOfOneWithAStopAsThatShipment(t *testing.T) {
	t.Parallel()

	failures := &fakeFailureDecider{}
	tool := newEvaluateServiceFailuresTool(failures).(*evaluateServiceFailuresTool)
	shipmentID, stopID := pulid.MustNew("shp_"), pulid.MustNew("stp_")
	params := map[string]any{
		"shipmentIds": []any{shipmentID.String()},
		paramStopID:   stopID.String(),
	}

	resolved, err := tool.ResolveSelection(t.Context(), executeParams(params))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"shipmentId": shipmentID.String(),
		paramStopID:  stopID.String(),
	}, resolved)

	require.NoError(t, tool.Execute(t.Context(), executeParams(params)))
	require.NotNil(t, failures.evaluatedStop)
	assert.Equal(t, shipmentID, failures.evaluatedStop.ShipmentID)
	assert.Equal(t, stopID, failures.evaluatedStop.StopID)
}

func TestEvaluateServiceFailures_LeavesASeveralShipmentCheckAsItIs(t *testing.T) {
	t.Parallel()

	tool := newEvaluateServiceFailuresTool(&fakeFailureDecider{}).(*evaluateServiceFailuresTool)
	params := map[string]any{"shipmentIds": manyIDs(2)}

	resolved, err := tool.ResolveSelection(t.Context(), executeParams(params))
	require.NoError(t, err)
	assert.Equal(t, params, resolved)
}

func TestEvaluateServiceFailures_ARefusalForBothScopesShowsBothCalls(t *testing.T) {
	t.Parallel()

	tool := newEvaluateServiceFailuresTool(&fakeFailureDecider{}).(*evaluateServiceFailuresTool)
	params := executeParams(map[string]any{
		"shipmentId":  pulid.MustNew("shp_").String(),
		"shipmentIds": manyIDs(2),
	})

	resolved, err := tool.ResolveSelection(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, params.Params, resolved, "Validate is left to refuse it")

	err = tool.Validate(t.Context(), params)
	require.ErrorIs(t, err, errBothEvaluationScopes)
	assert.Contains(t, err.Error(), "send shipmentId without shipmentIds")
	assert.Contains(t, err.Error(), "send shipmentIds without shipmentId")
}
