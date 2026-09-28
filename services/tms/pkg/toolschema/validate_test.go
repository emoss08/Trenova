package toolschema

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldErrors(t *testing.T, err error) map[string]string {
	t.Helper()

	require.Error(t, err)
	multiErr, ok := err.(*errortypes.MultiError)
	require.True(t, ok, "expected field-level errors, got %T: %v", err, err)

	out := make(map[string]string, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		out[entry.Field] = entry.Message
	}

	return out
}

func TestValidate_AcceptsArgumentsThatFit(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Validate(notifyDriverSchema(), map[string]any{
		"workerId": "wrk_1", "title": "Delivery moved", "message": "Be there at 3.",
		"priority": "high", "withinDays": 3, "codes": []any{"A"},
	}))
}

// Each thing wrong is filed under the field it is wrong in, so a form can
// put the message beside the value rather than at the top.
func TestValidate_FilesEachProblemUnderItsField(t *testing.T) {
	t.Parallel()

	errs := fieldErrors(t, Validate(notifyDriverSchema(), map[string]any{
		"workerId":   "wrk_1",
		"title":      "x",
		"priority":   "urgent",
		"withinDays": 0,
		"urgent":     "yes",
		"bogus":      1,
	}))

	assert.Contains(t, errs, "message", "a missing required value is filed under its own name")
	assert.Equal(t, "This value is required", errs["message"])
	assert.Contains(t, errs, "priority")
	assert.Contains(t, errs, "withinDays")
	assert.Contains(t, errs, "urgent")
	assert.Equal(t, "This tool does not take this value", errs["bogus"])
	assert.NotContains(t, errs, "params", "the root's summary is not repeated")
}

func TestValidate_TakesGoNumbersAndSchemaLists(t *testing.T) {
	t.Parallel()

	// Arguments and schemas written in Go hold ints and []string where the
	// validator wants floats and []any; both are normalized on the way in.
	assert.NoError(t, Validate(map[string]any{
		"type":       "object",
		"properties": map[string]any{"n": map[string]any{"type": "integer", "minimum": 1}},
		"required":   []string{"n"},
	}, map[string]any{"n": int64(2)}))
}

func TestValidate_AcceptsAnythingForAnEmptySchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Validate(nil, map[string]any{"anything": true}))
}

// The compiled schema is kept by the tool's name: the second call under a
// name is judged by the schema compiled for the first, whatever it passes.
func TestValidator_KeepsTheCompiledSchemaPerToolName(t *testing.T) {
	t.Parallel()

	validator := NewValidator()
	strict := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"n": map[string]any{"type": "integer"}},
		"additionalProperties": false,
	}
	loose := map[string]any{"type": "object"}

	require.Error(t, validator.ValidateFor("count", strict, map[string]any{"extra": 1}))
	assert.Error(t, validator.ValidateFor("count", loose, map[string]any{"extra": 1}),
		"the schema compiled first for this name is the one that judges")
	assert.NoError(t, validator.ValidateFor("other", loose, map[string]any{"extra": 1}),
		"another name compiles its own")
	assert.NoError(t, validator.ValidateFor("count", strict, map[string]any{"n": 2}))

	var none *Validator
	assert.Error(t, none.ValidateFor("count", strict, map[string]any{"extra": 1}),
		"a nil validator still validates, without keeping anything")
	assert.NoError(t, validator.ValidateFor("empty", nil, map[string]any{"extra": 1}))
}

func TestValidator_FilesNestedProblemsUnderTheirPaths(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipment": map[string]any{
				"type": "object",
				"properties": map[string]any{
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
											"sequence": map[string]any{"type": "integer"},
										},
										"required":             []string{"locationId", "type"},
										"additionalProperties": false,
									},
								},
							},
							"required":             []string{"stops"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"moves"},
				"additionalProperties": false,
			},
		},
		"required":             []string{"shipment"},
		"additionalProperties": false,
	}

	errs := fieldErrors(t, NewValidator().ValidateFor("create_shipment", schema, map[string]any{
		"shipment": map[string]any{
			"moves": []any{map[string]any{
				"type": "Linehaul",
				"stops": []any{map[string]any{
					"type":     "Pickup",
					"sequence": "first",
				}},
			}},
		},
	}))

	assert.Equal(t, "This tool does not take this value", errs["shipment.moves[0].type"])
	assert.Equal(t, "This value is required", errs["shipment.moves[0].stops[0].locationId"])
	assert.Contains(t, errs, "shipment.moves[0].stops[0].sequence")
}

// The source keyword is for people and tests; a strict model endpoint
// refuses a keyword it does not know, at any depth.
func TestForModel_StripsTheEnumSourceKeyword(t *testing.T) {
	t.Parallel()

	shown := ForModel(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"priority": map[string]any{
				"type":    "string",
				"enum":    []string{"Low", "High"},
				KeyEnumOf: "shipment.commentPriority",
			},
			"lines": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"channel": map[string]any{
							"type":    "string",
							"enum":    []string{"Email"},
							KeyEnumOf: "tender.channel",
						},
					},
				},
			},
		},
	})

	priority, _ := shown["properties"].(map[string]any)["priority"].(map[string]any)
	assert.NotContains(t, priority, KeyEnumOf)
	assert.Equal(t, []string{"Low", "High"}, priority["enum"])

	lines, _ := shown["properties"].(map[string]any)["lines"].(map[string]any)
	items, _ := lines["items"].(map[string]any)
	channel, _ := items["properties"].(map[string]any)["channel"].(map[string]any)
	assert.NotContains(t, channel, KeyEnumOf)
}

func TestChanged_KeepsOnlyWhatDiffers(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{
		"title":      "A",
		"withinDays": float64(3),
		"codes":      []any{"x"},
		"note":       nil,
	}
	changed := Changed(proposed, map[string]any{
		"title":      "A",
		"withinDays": 3,
		"codes":      []any{"x", "y"},
		"priority":   "high",
		"missing":    nil,
		"note":       nil,
	})

	assert.Equal(t, map[string]any{"codes": []any{"x", "y"}, "priority": "high"}, changed)
}
