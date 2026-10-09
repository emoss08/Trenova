package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coercionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit":    map[string]any{"type": "integer"},
			"amount":   map[string]any{"type": "string", "description": "A decimal such as 12.50."},
			"shared":   map[string]any{"type": "boolean"},
			"priority": map[string]any{"type": "string", "enum": []string{"Low", "High"}},
			"ids": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"stops": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"sequence": map[string]any{"type": "integer"},
						"type":     map[string]any{"type": "string", "enum": []string{"Pickup", "Delivery"}},
					},
				},
			},
			"note": map[string]any{"type": "string"},
			"customerId": map[string]any{
				"type":        "string",
				"description": "The customer, from list_customers. Never guess one.",
			},
		},
		"required":             []string{"limit"},
		"additionalProperties": false,
	}
}

/*
A model sent "30" for an integer, 12.5 for a text amount, "high" for High,
one id where a list was declared and the stops' sequence as text. Each was
refused on its own and cost a round trip; a model refused twice told the
person the tool did not work. Every one of those readings is certain, so the
runtime makes it and says so.
*/
func TestCoerceArguments_ReadsValuesAsTheToolDeclaresThem(t *testing.T) {
	t.Parallel()

	sent := map[string]any{
		"limit":    "30",
		"amount":   12.5,
		"shared":   "true",
		"priority": "high",
		"ids":      "shp_1",
		"stops": []any{
			map[string]any{"sequence": "0", "type": "pickup"},
		},
		"note": nil,
		"NOTE": "late",
	}

	got, coercions := coerceArguments(coercionSchema(), sent)

	assert.Equal(t, map[string]any{
		"limit":    float64(30),
		"amount":   "12.5",
		"shared":   true,
		"priority": "High",
		"ids":      []any{"shp_1"},
		"stops": []any{
			map[string]any{"sequence": float64(0), "type": "Pickup"},
		},
		"note": "late",
	}, got)
	assert.Equal(t, "30", sent["limit"], "the model's own call is never changed")
	assert.NotEmpty(t, coercions)

	note := coercionNote(coercions)
	assert.Contains(t, note, `limit "30" was read as the number 30`)
	assert.Contains(t, note, "Send them that way from now on.")
	assert.Contains(t, note, "and 3 more", "the note names five and counts the rest")
}

func TestCoerceArguments_SplitsCommaSeparatedTextAndUnwrapsAWrappedList(t *testing.T) {
	t.Parallel()

	got, _ := coerceArguments(coercionSchema(), map[string]any{
		"limit": 1, "ids": "shp_1, shp_2,",
	})
	assert.Equal(t, []any{"shp_1", "shp_2"}, got["ids"])

	got, _ = coerceArguments(coercionSchema(), map[string]any{
		"limit": 1, "ids": map[string]any{"item": []any{"shp_1"}},
	})
	assert.Equal(t, []any{"shp_1"}, got["ids"])
}

// Anything less than certain is left for the schema to refuse by name: a
// word where a number was declared, a value outside the enum, a null for a
// required parameter.
func TestCoerceArguments_LeavesUncertainValuesForTheSchema(t *testing.T) {
	t.Parallel()

	sent := map[string]any{"limit": "thirty", "priority": "Critical", "shared": "maybe"}

	got, coercions := coerceArguments(coercionSchema(), sent)

	assert.Equal(t, sent, got)
	assert.Empty(t, coercions)

	_, err := contractCall(toolschema.NewValidator(), "stub", coercionSchema(),
		map[string]any{"limit": nil})
	require.Error(t, err, "a null required parameter is still missing")
}

// A pro number sent where a record id was wanted used to reach the service
// and come back "not found", which the model read as the record not
// existing. The shape is checked first, and the refusal says where the id
// comes from.
func TestContractCall_RefusesAValueThatIsNotARecordId(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", coercionSchema(),
		map[string]any{"limit": 1, "customerId": "Acme Foods"})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := argumentProblems(multiErr)
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], `customerId: "Acme Foods" is not a record id`)
	assert.Contains(t, problems[0], "from list_customers")

	_, err = contractCall(toolschema.NewValidator(), "stub", coercionSchema(),
		map[string]any{"limit": 1, "customerId": "cus_01J9Z3QK8M5T7V2X4Y6W0R1N8P"})
	require.NoError(t, err, "a value shaped like a record id passes")
}

func typedIDSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": agenttoolschema.RecordID(permission.ResourceCustomer, "The customer",
				"list_customers"),
			"invoiceIds": agenttoolschema.RecordIDs(permission.ResourceInvoice,
				"The invoices, from list_invoices.", 10),
			"stops": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": agenttoolschema.RecordID(permission.ResourceLocation,
							"The stop's location", "list_locations"),
					},
				},
			},
		},
	}
}

// An id of the wrong kind passed the shape check and came back "not found",
// which the model read as the record not existing. A parameter marked with
// its resource is held to that resource's prefix, wherever it sits, and the
// refusal names what the id is and where the right one comes from.
func TestContractCall_RefusesAnIDOfAnotherKind(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", typedIDSchema(),
		map[string]any{
			"customerId": "car_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
			"invoiceIds": []any{"inv_01J9Z3QK8M5T7V2X4Y6W0R1N8P", "shp_01J9Z3QK8M5T7V2X4Y6W0R1N8P"},
			"stops": []any{
				map[string]any{"location": "zzz_01J9Z3QK8M5T7V2X4Y6W0R1N8P"},
			},
		})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := argumentProblems(multiErr)
	require.Len(t, problems, 3)
	assert.Contains(t, problems[0], `customerId: "car_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is a carrier `+
		`id; this parameter takes a customer id, from list_customers.`)
	assert.Contains(t, problems[1], `invoiceIds[1]: "shp_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is a `+
		`shipment id; this parameter takes an invoice id, from list_invoices.`)
	assert.Contains(t, problems[2], `stops[0].location: "zzz_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is `+
		`not a location id: location ids start with "loc_".`)
}

// The mark decides, not the parameter's name: "location" carries no Id
// suffix and is still checked, and a right id of the right kind passes.
func TestContractCall_AcceptsAnIDOfTheMarkedKind(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", typedIDSchema(),
		map[string]any{
			"customerId": "cus_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
			"invoiceIds": []any{"inv_01J9Z3QK8M5T7V2X4Y6W0R1N8P"},
			"stops": []any{
				map[string]any{"location": "loc_01J9Z3QK8M5T7V2X4Y6W0R1N8P"},
			},
		})

	require.NoError(t, err)

	_, err = contractCall(toolschema.NewValidator(), "stub", typedIDSchema(),
		map[string]any{"stops": []any{map[string]any{"location": "Dallas yard"}}})
	require.Error(t, err, "a value that is no id at all is refused under the mark too")
}

func TestIDSource_ReadsWhereTheIDComesFrom(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "list_customers", idSource("The customer, from list_customers. Never guess one."))
	assert.Equal(t, "get_shipment, search_shipments or the page you are on",
		idSource("The shipment, from get_shipment, search_shipments or the page you are on. "+
			"Never guess one."))
	assert.Empty(t, idSource("A note."))
}

// A refusal names what the tool wanted at each path, read from the schema,
// so one corrected call follows instead of a guess.
func TestArgumentOutcome_DescribesWhatEachRefusedParameterTakes(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", coercionSchema(),
		map[string]any{"limit": "thirty", "priority": "Critical", "extra": 1,
			"stops": []any{map[string]any{"type": "Drop"}}})
	require.Error(t, err)

	outcome := argumentOutcome("stub", coercionSchema(), err)

	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.content, "was not run or proposed")
	assert.Contains(t, outcome.content, "- limit: got string, want integer (integer)")
	assert.Contains(t, outcome.content, "- priority: value must be one of 'Low', 'High' (string; one of Low, High)")
	assert.Contains(t, outcome.content, "- stops[0].type: value must be one of 'Pickup', 'Delivery' (string; one of Pickup, Delivery)")
	assert.Contains(t, outcome.content, "- extra: This tool does not take this value (this tool takes: amount, customerId, ids, limit, note, priority, shared, stops)")
	assert.Contains(t, outcome.content, "changing only what is named")
}

func datedSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"startDate": agenttoolschema.Date("The first day."),
			"pickupAt":  agenttoolschema.LocalDateTime("When pickup opens."),
			"happened":  agenttoolschema.DateTime("When it happened."),
		},
	}
}

// Midnight UTC sent for a day is that day, and a local time written with a
// space or with seconds is the minute it names: each reading is certain, is
// told back, and reaches the tool in the declared shape.
func TestContractCall_ReadsDatesWhereTheReadingIsCertain(t *testing.T) {
	t.Parallel()

	contract, err := contractCall(toolschema.NewValidator(), "stub", datedSchema(),
		map[string]any{
			"startDate": "2026-10-01T00:00:00Z",
			"pickupAt":  "2026-10-01 08:00:30",
		})
	require.NoError(t, err)

	assert.Equal(t, "2026-10-01", contract.args["startDate"])
	assert.Equal(t, "2026-10-01T08:00", contract.args["pickupAt"])
	assert.Contains(t, contract.note(), `startDate "2026-10-01T00:00:00Z" was read as the day 2026-10-01`)
	assert.Contains(t, contract.note(), `pickupAt "2026-10-01 08:00:30" was read as 2026-10-01T08:00`)
}

// A time that is not midnight is not certainly a day, and a number is never
// read as a date: it is refused as the Unix time it is, with the shape the
// parameter takes.
func TestContractCall_RefusesDatesItCannotReadForCertain(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", datedSchema(),
		map[string]any{
			"startDate": "2026-10-01T14:00:00Z",
			"pickupAt":  float64(1790812800),
			"happened":  "1790812800",
		})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := argumentProblems(multiErr)
	require.Len(t, problems, 3)
	assert.Contains(t, problems[0], "happened: 1790812800 is a Unix time; send a date-time")
	assert.Contains(t, problems[1], "pickupAt: 1790812800 is a Unix time; send a local-date-time "+
		"as YYYY-MM-DDTHH:MM with no UTC offset, such as 2026-10-01T08:00")
	assert.Contains(t, problems[2], `startDate: "2026-10-01T14:00:00Z" is not a date; send `+
		`YYYY-MM-DD, such as 2026-10-01`)
}
