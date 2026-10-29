package argcheck

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shipmentEntrySchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipment": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customerId": map[string]any{"type": "string"},
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
											"type": map[string]any{
												"type": "string",
												"enum": []string{"Pickup", "Delivery"},
											},
											"sequence": map[string]any{
												"type": "integer",
											},
											"scheduledWindowStart": map[string]any{
												"type": "integer",
											},
										},
										"required": []string{
											"locationId", "type", "sequence", "scheduledWindowStart",
										},
										"additionalProperties": false,
									},
								},
							},
							"required":             []string{"stops"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"customerId", "moves"},
				"additionalProperties": false,
			},
		},
		"required":             []string{"shipment"},
		"additionalProperties": false,
	}
}

// The transcript this guards against: moves[].type is not a field and was
// dropped without a word, a stop type the domain does not have, a sequence
// as a word and a window as text all reached the tool. Each is refused at
// its own path, in one answer, so the model fixes the call once.
func TestContractArguments_RefusesEachProblemAtItsPath(t *testing.T) {
	t.Parallel()

	_, _, err := ContractArguments(toolschema.NewValidator(), "shipment_create",
		shipmentEntrySchema(), map[string]any{
			"shipment": map[string]any{
				"customerId": "cust_1",
				"moves": []any{map[string]any{
					"type": "Linehaul",
					"stops": []any{
						map[string]any{
							"locationId":           "loc_1",
							"type":                 "Drop",
							"sequence":             "first",
							"scheduledWindowStart": 1790000000,
						},
						map[string]any{
							"type":                 "Delivery",
							"sequence":             1,
							"scheduledWindowStart": "2026-09-28T09:00:00Z",
						},
					},
				}},
			},
		})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := Problems(multiErr)
	byPath := make(map[string]string, len(problems))
	for _, line := range problems {
		path, message, _ := strings.Cut(line, ": ")
		byPath[path] = message
	}

	assert.Equal(t, "This tool does not take this value", byPath["shipment.moves[0].type"])
	assert.Contains(t, byPath, "shipment.moves[0].stops[0].type", "a value outside the enum")
	assert.Contains(t, byPath, "shipment.moves[0].stops[0].sequence", "a word for an integer")
	assert.Equal(t, "This value is required", byPath["shipment.moves[0].stops[1].locationId"])
	assert.Contains(t, byPath, "shipment.moves[0].stops[1].scheduledWindowStart",
		"a text where an integer was declared")
	assert.Len(t, problems, 5, "nothing else is wrong with the call")
}

func TestContractArguments_StillAliasesAMisnamedRequiredParameter(t *testing.T) {
	t.Parallel()

	schema := runtimetest.ClosedSchema("shipmentId")
	schema["required"] = []string{"shipmentId"}
	sent := map[string]any{"shipment_id": "shp_1"}

	got, aliases, err := ContractArguments(toolschema.NewValidator(), "shipment_get", schema, sent)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, got)
	assert.Equal(t, []argumentAlias{{From: "shipment_id", To: "shipmentId"}}, aliases)
	assert.Equal(t, map[string]any{"shipment_id": "shp_1"}, sent,
		"the model's own call is never changed")
}

func TestContractArguments_AnOpenSchemaAcceptsAnything(t *testing.T) {
	t.Parallel()

	got, _, err := ContractArguments(nil, "stub", map[string]any{"type": "object"},
		map[string]any{"anything": 1, agenttoolports.SelfScopeOwnerParam: "usr_x"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"anything": 1}, got,
		"the owner key is set aside whatever the schema says")
}

func TestArgumentOutcome_ReportsAnUncheckableSchemaAsAFailure(t *testing.T) {
	t.Parallel()

	_, _, err := ContractArguments(nil, "broken", map[string]any{"type": 12}, map[string]any{})
	require.Error(t, err)

	outcome := Refusal("broken", map[string]any{"type": 12}, err)
	assert.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "could not be checked")
	assert.NotContains(t, outcome.Content, "Fix the call")
}
