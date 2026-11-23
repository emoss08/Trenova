package argcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	shipmentID = pulid.ID("shp_01J9Z3QK8M5T7V2X4Y6W0R1N8P")
	locationID = pulid.ID("loc_01J9Z3QK8M5T7V2X4Y6W0R1N8P")
)

var shipmentKind = recordkind.Kind(permission.ResourceShipment)

func numberSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentIds": agenttoolschema.RecordIDs(permission.ResourceShipment,
				"The shipments, from shipment_list.", 25),
			"stops": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": agenttoolschema.RecordID(permission.ResourceLocation,
							"The stop's location", "location_list"),
					},
				},
			},
			"customerId": map[string]any{
				"type":        "string",
				"description": "The customer, from customer_list. Never guess one.",
			},
		},
	}
}

func refusedPaths(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	return Problems(multiErr)
}

/*
The owner asked to see two loads side by side by their numbers, the model sent
the numbers where shipment_get takes ids, and every call was refused before it
ran. A number on a parameter marked with the kinds it takes is set aside to be
read as its record, wherever it sits, instead of refused.
*/
func TestContractCall_SetsAsideANumberSentForAMarkedID(t *testing.T) {
	t.Parallel()

	contract, err := ContractCall(toolschema.NewValidator(), "stub", numberSchema(),
		map[string]any{
			"shipmentIds": []any{"SEED-SHP-008", shipmentID.String()},
			"stops":       []any{map[string]any{"location": " TERM-LA "}},
		})

	require.NoError(t, err)
	numbers := contract.Numbers()
	require.Len(t, numbers, 2)
	assert.Equal(t, "shipmentIds[0]", numbers[0].Path)
	assert.Equal(t, "SEED-SHP-008", numbers[0].Number)
	assert.Equal(t, []recordkind.Kind{shipmentKind}, numbers[0].Kinds)
	assert.Equal(t, "stops[0].location", numbers[1].Path)
	assert.Equal(t, "TERM-LA", numbers[1].Number, "the number is read trimmed")
}

// What was refused before is refused the same way: a mis-copied id, an id of
// another kind, a value on an unmarked parameter, words with a space, and a
// number in a call refused for something else.
func TestContractCall_StillRefusesWhatIsNoNumber(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args map[string]any
		want []string
	}{
		"mis-copied id": {
			args: map[string]any{"shipmentIds": []any{"shp_01J9Z3QK8M5T7V2X4Y6W0R1N8"}},
			want: []string{`shipmentIds[0]: "shp_01J9Z3QK8M5T7V2X4Y6W0R1N8" is not a whole id`},
		},
		"id of another kind": {
			args: map[string]any{"shipmentIds": []any{"inv_01J9Z3QK8M5T7V2X4Y6W0R1N8P"}},
			want: []string{`shipmentIds[0]: "inv_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is an invoice id`},
		},
		"unmarked parameter": {
			args: map[string]any{"customerId": "ACME"},
			want: []string{`customerId: "ACME" is not a record id`},
		},
		"words with a space": {
			args: map[string]any{"stops": []any{map[string]any{"location": "Dallas yard"}}},
			want: []string{`stops[0].location: "Dallas yard" is not a record id`},
		},
		"number beside another problem": {
			args: map[string]any{"shipmentIds": []any{"SEED-SHP-008"}, "customerId": "ACME"},
			want: []string{
				`customerId: "ACME" is not a record id`,
				`shipmentIds[0]: "SEED-SHP-008" is not a record id`,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ContractCall(toolschema.NewValidator(), "stub", numberSchema(), tc.args)
			problems := refusedPaths(t, err)
			require.Len(t, problems, len(tc.want))
			for idx, want := range tc.want {
				assert.Contains(t, problems[idx], want)
			}
		})
	}
}

func TestReadNumbers_PutsEachRecordWhereItsNumberWasAndSaysSo(t *testing.T) {
	t.Parallel()

	sentStops := []any{map[string]any{"location": "TERM-LA"}}
	sent := map[string]any{
		"shipmentIds": []any{"SEED-SHP-008", shipmentID.String()},
		"stops":       sentStops,
	}
	contract, err := ContractCall(toolschema.NewValidator(), "stub", numberSchema(), sent)
	require.NoError(t, err)

	other := pulid.ID("shp_01J9Z3QK8M5T7V2X4Y6W0R1N8Q")
	read, err := contract.ReadNumbers(map[string][]NumberCandidate{
		"shipmentIds[0]": {{
			Kind: shipmentKind, ID: other, NumberName: "PRO number", Number: "SEED-SHP-008",
		}},
		"stops[0].location": {{
			Kind:       recordkind.Kind(permission.ResourceLocation),
			ID:         locationID,
			NumberName: "location code",
			Number:     "TERM-LA",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"shipmentIds": []any{other.String(), shipmentID.String()},
		"stops":       []any{map[string]any{"location": locationID.String()}},
	}, read.Args)
	assert.Equal(t, "SEED-SHP-008", sent["shipmentIds"].([]any)[0], "the model's call is kept")
	assert.Equal(t, "TERM-LA", sentStops[0].(map[string]any)["location"])
	assert.Empty(t, read.Numbers())
	assert.Equal(t,
		`Record numbers were read as the records they name: shipmentIds[0] "SEED-SHP-008" is `+
			`the shipment with that PRO number, id `+other.String()+`; stops[0].location `+
			`"TERM-LA" is the location with that location code, id `+locationID.String()+
			`. A number is not a record id: send the id from now on.`,
		read.Note(),
	)
}

// A number no record carries is refused as it always was; one that more than
// one record carries is refused naming each, so the model asks which.
func TestReadNumbers_RefusesANumberThatIsNotExactlyOneRecord(t *testing.T) {
	t.Parallel()

	contract, err := ContractCall(toolschema.NewValidator(), "stub", numberSchema(),
		map[string]any{"shipmentIds": []any{"SEED-SHP-008", "S26110493143683"}})
	require.NoError(t, err)

	_, err = contract.ReadNumbers(map[string][]NumberCandidate{
		"shipmentIds[1]": {
			{Kind: shipmentKind, ID: shipmentID, NumberName: "PRO number", Number: "S26110493143683"},
			{
				Kind:       recordkind.Kind(permission.ResourceInvoice),
				ID:         pulid.ID("inv_01J9Z3QK8M5T7V2X4Y6W0R1N8P"),
				NumberName: "invoice number",
				Number:     "S26110493143683",
			},
		},
	})

	problems := refusedPaths(t, err)
	require.Len(t, problems, 2)
	assert.Contains(t, problems[0], `shipmentIds[0]: "SEED-SHP-008" is not a record id. `+
		`A record id looks like shp_01J…`)
	assert.Equal(t, `shipmentIds[1]: "S26110493143683" is not a record id, and more than one `+
		`record carries it as its number: the shipment with PRO number S26110493143683 or the `+
		`invoice with invoice number S26110493143683. Ask the person which one they mean, then `+
		`send that record's id.`, problems[1])
}
