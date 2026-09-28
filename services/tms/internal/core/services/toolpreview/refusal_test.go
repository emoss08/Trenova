package toolpreview

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func duplicateBOL() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"bol",
		errortypes.ErrInvalid,
		"BOL is already in use by shipment(s) with Pro Number(s): {0}",
		"SEED-DET-009",
	)
	multiErr.WithPrefix("moves").WithIndex("stops", 1).Add(
		"locationId",
		errortypes.ErrRequired,
		"Location is required",
	)

	return multiErr
}

func shipmentSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipment": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bol": map[string]any{"type": "string"},
					"moves": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"stops": map[string]any{
									"type": "array",
									"items": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"locationId": map[string]any{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			"sourceDocumentId": map[string]any{"type": "string"},
		},
	}
}

func TestWouldFail_CarriesEachProblemOfAValidationRefusal(t *testing.T) {
	t.Parallel()

	warning := WouldFail(fmt.Errorf("create shipment: %w", duplicateBOL()))

	assert.Equal(t, agent.PreviewWarningWouldFail, warning.Code)
	assert.Equal(t,
		"This would be refused as it stands: create shipment: validation failed:\n"+
			"- BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009\n"+
			"- Location is required",
		warning.Message,
		"the message keeps the whole refusal as it was worded",
	)
	require.Len(t, warning.Args, 1, "the refusal is an argument, so two refusals stay apart")
	require.Len(t, warning.Reasons, 2)
	assert.Equal(t, agent.PreviewReason{
		Field:   "bol",
		Label:   "BOL",
		Message: "BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
	}, warning.Reasons[0])
	assert.Equal(t, agent.PreviewReason{
		Field:   "moves.stops[1].locationId",
		Label:   "Location ID",
		Message: "Location is required",
	}, warning.Reasons[1])
}

func TestWouldFail_ABusinessRefusalIsOneReasonWithoutAField(t *testing.T) {
	t.Parallel()

	warning := WouldFail(errortypes.NewBusinessError("The invoice is already posted"))

	require.Len(t, warning.Reasons, 1)
	assert.Equal(t, agent.PreviewReason{Message: "The invoice is already posted"},
		warning.Reasons[0])
}

func TestLocateReasons_NamesTheParameterThatCarriesEachField(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add("bol", errortypes.ErrInvalid, "BOL is already in use")
	multiErr.WithPrefix("moves[0]").WithIndex("stops", 1).
		Add("locationId", errortypes.ErrRequired, "Location is required")
	multiErr.Add("sourceDocumentId", errortypes.ErrInvalid, "Not an id")
	multiErr.Add("ratingMethod", errortypes.ErrInvalid, "Not a parameter at all")

	warnings := []agent.PreviewWarning{WouldFail(multiErr)}
	LocateReasons(warnings, shipmentSchema())

	reasons := warnings[0].Reasons
	require.Len(t, reasons, 4)
	assert.Equal(t, "shipment.bol", reasons[0].Param,
		"a field of the record the call carries whole is found inside it")
	assert.Equal(t, "shipment.moves[0].stops[1].locationId", reasons[1].Param)
	assert.Equal(t, "sourceDocumentId", reasons[2].Param, "a top-level parameter is itself")
	assert.Empty(t, reasons[3].Param, "a field the call does not carry names no parameter")
}

func TestLocateReasons_LeavesAParameterTheToolNamedAlone(t *testing.T) {
	t.Parallel()

	warnings := []agent.PreviewWarning{{
		Code: agent.PreviewWarningWouldFail,
		Reasons: []agent.PreviewReason{
			{Field: "bol", Message: "Taken", Param: "reference"},
		},
	}, {
		Code:    agent.PreviewWarningTargetChanged,
		Reasons: []agent.PreviewReason{{Field: "bol", Message: "not a refusal"}},
	}}
	LocateReasons(warnings, shipmentSchema())

	assert.Equal(t, "reference", warnings[0].Reasons[0].Param)
	assert.Empty(t, warnings[1].Reasons[0].Param, "only a would_fail warning is located")
}
