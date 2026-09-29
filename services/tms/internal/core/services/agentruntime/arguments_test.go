package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func closedSchema(fields ...string) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
}

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
											"sequence":             map[string]any{"type": "integer"},
											"scheduledWindowStart": map[string]any{"type": "integer"},
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

	_, _, err := contractArguments(toolschema.NewValidator(), "create_shipment",
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
	problems := argumentProblems(multiErr)
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

	schema := closedSchema("shipmentId")
	schema["required"] = []string{"shipmentId"}
	sent := map[string]any{"shipment_id": "shp_1"}

	got, aliases, err := contractArguments(toolschema.NewValidator(), "get_shipment", schema, sent)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, got)
	assert.Equal(t, []argumentAlias{{From: "shipment_id", To: "shipmentId"}}, aliases)
	assert.Equal(t, map[string]any{"shipment_id": "shp_1"}, sent,
		"the model's own call is never changed")
}

func TestContractArguments_AnOpenSchemaAcceptsAnything(t *testing.T) {
	t.Parallel()

	got, _, err := contractArguments(nil, "stub", map[string]any{"type": "object"},
		map[string]any{"anything": 1, serviceports.SelfScopeOwnerParam: "usr_x"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"anything": 1}, got,
		"the owner key is set aside whatever the schema says")
}

func TestArgumentOutcome_ReportsAnUncheckableSchemaAsAFailure(t *testing.T) {
	t.Parallel()

	_, _, err := contractArguments(nil, "broken", map[string]any{"type": 12}, map[string]any{})
	require.Error(t, err)

	outcome := argumentOutcome("broken", err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.content, "could not be checked")
	assert.NotContains(t, outcome.content, "Fix the call")
}

// The model sent an owner for a write on the person's own records. The key
// is not a parameter the tool declares, so a closed schema would refuse it;
// it is the runtime's, set aside before the check and stamped from the
// person in the conversation after.
func TestRun_StripsTheOwnerTheModelSentAndStampsThePerson(t *testing.T) {
	t.Parallel()

	stub := actionTool("add_home_widget", agent.TierPropose, nil)
	stub.Resource = permission.ResourceHomeLayoutPreset
	stub.Schema = closedSchema("key")
	tool := selfScopedAction{stub}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("add_home_widget", map[string]any{
			"key":                            "kpi",
			serviceports.SelfScopeOwnerParam: pulid.MustNew("usr_").String(),
		}),
		textTurn("Proposed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	actor := testActor()

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("add_home_widget"),
		Actor:      actor,
		Input:      "put on-time on my dashboard",
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 1, "the call fit the schema once the owner was set aside")
	assert.Equal(t, map[string]any{
		"key":                            "kpi",
		serviceports.SelfScopeOwnerParam: actor.UserID.String(),
	}, result.Actions[0].Arguments)
}

// A read is held to its schema as a write is. A list tool given a filter it
// does not declare used to list everything, and the model read the answer
// as if the filter had applied.
func TestRun_ValidatesAQueryToolsArguments(t *testing.T) {
	t.Parallel()

	tool := &agentruntimetest.StubQueryTool{
		ToolName: "list_shipments",
		Result:   map[string]any{"items": []any{}},
		Schema:   closedSchema("status"),
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("list_shipments", map[string]any{"status": "New", "customer": "Acme"}),
		textTurn("I could not list them."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("list_shipments"),
		Actor:      testActor(),
		Input:      "Acme's new loads",
	})
	require.NoError(t, err)

	assert.Zero(t, tool.Calls, "the read never ran")
	assert.Equal(t, 1, result.ToolCallsUsed)
	refusals := toolMessages(result)
	require.Len(t, refusals, 1)
	assert.True(t, refusals[0].ToolFailed)
	requireRefusalNames(t, refusals[0].Content, []string{"customer"})
}
