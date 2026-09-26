package toolschema

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func transferSchema() map[string]any {
	return map[string]any{
		KeyType: TypeObject,
		KeyProperties: map[string]any{
			"shipmentIds": RecordSubset("shipment", map[string]any{
				KeyType:        TypeArray,
				KeyDescription: "The shipments to transfer.",
				KeyItems:       map[string]any{KeyType: TypeString},
			}),
			"billType": map[string]any{KeyType: TypeString, KeyEnum: []string{"Invoice"}},
		},
		KeyRequired: []string{"shipmentIds"},
	}
}

func TestFields_ARecordSubsetNamesTheResourceItDrawsFrom(t *testing.T) {
	t.Parallel()

	fields := Fields(transferSchema())
	require.Len(t, fields, 2)

	subset := fields[0]
	assert.Equal(t, "shipmentIds", subset.Name)
	assert.Equal(t, KindRecordSubset, subset.Kind)
	assert.Equal(t, "shipment", subset.Resource)
	assert.Empty(t, fields[1].Resource, "only a subset field names a resource")
}

func TestCheckSubsets_AnApproverMayNarrowTheRecords(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{"shipmentIds": []any{"shp_a", "shp_b", "shp_c"}}
	changed := map[string]any{"shipmentIds": []any{"shp_c", "shp_a"}}

	require.NoError(t, CheckSubsets(transferSchema(), proposed, changed))
}

func TestCheckSubsets_AnApproverMayNotAddARecord(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{"shipmentIds": []any{"shp_a", "shp_b"}}
	changed := map[string]any{"shipmentIds": []any{"shp_a", "shp_z"}}

	err := CheckSubsets(transferSchema(), proposed, changed)
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Contains(t, err.Error(), "shp_z")
	assert.Equal(t, "shipmentIds", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrForbidden, multiErr.Errors[0].Code)
}

func TestCheckSubsets_AnApproverMustKeepAtLeastOneRecord(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{"shipmentIds": []any{"shp_a"}}

	for name, changed := range map[string]map[string]any{
		"empty list": {"shipmentIds": []any{}},
		"cleared":    {"shipmentIds": nil},
		"not a list": {"shipmentIds": "shp_a"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, CheckSubsets(transferSchema(), proposed, changed))
		})
	}
}

func TestCheckSubsets_LeavesOtherParametersAlone(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{"shipmentIds": []any{"shp_a"}, "billType": "Invoice"}
	changed := map[string]any{"billType": "Invoice"}

	require.NoError(t, CheckSubsets(transferSchema(), proposed, changed))
	require.NoError(t, CheckSubsets(notifyDriverSchema(), proposed, changed))
}

// The key that marks a subset is ours, not JSON Schema's. A provider that
// validates schemas strictly refuses a keyword it does not know, so what a
// model is shown carries none of them, at any depth, and the parameter names
// they sit beside survive.
func TestForModel_StripsExtensionKeywordsAtEveryDepth(t *testing.T) {
	t.Parallel()

	schema := transferSchema()
	schema[KeyProperties].(map[string]any)["x-named"] = map[string]any{KeyType: TypeString}
	schema[KeyProperties].(map[string]any)["nested"] = map[string]any{
		KeyType: TypeArray,
		KeyItems: map[string]any{
			KeyType:    TypeObject,
			"x-hidden": true,
			KeyProperties: map[string]any{
				"ids": RecordSubset("worker", map[string]any{KeyType: TypeArray}),
			},
		},
	}

	stripped := ForModel(schema)

	properties := stripped[KeyProperties].(map[string]any)
	assert.NotContains(t, properties["shipmentIds"], KeySubsetOf)
	assert.Contains(t, properties, "x-named", "a parameter's name is not a keyword")
	items := properties["nested"].(map[string]any)[KeyItems].(map[string]any)
	assert.NotContains(t, items, "x-hidden")
	assert.NotContains(t, items[KeyProperties].(map[string]any)["ids"], KeySubsetOf)

	original := schema[KeyProperties].(map[string]any)["shipmentIds"].(map[string]any)
	assert.Equal(t, "shipment", original[KeySubsetOf], "the tool's own schema is untouched")
}

func TestValidate_AcceptsASchemaCarryingTheSubsetKeyword(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(transferSchema(), map[string]any{
		"shipmentIds": []any{"shp_a"},
	}))
}

// A person unticks records from a list of what the agent proposed: the
// parameter as proposed, in its order, each record once, named by its id
// until a label is read. What is not an id is not listed.
func TestSubsetChoices_ListsEachProposedRecordOnceInOrder(t *testing.T) {
	t.Parallel()

	choices := SubsetChoices(transferSchema(), map[string]any{
		"shipmentIds": []any{"shp_b", " shp_a ", "shp_b", 7, "", nil},
		"billType":    "Invoice",
	})

	assert.Equal(t, map[string][]Choice{
		"shipmentIds": {{ID: "shp_b", Label: "shp_b"}, {ID: "shp_a", Label: "shp_a"}},
	}, choices)
}

func TestSubsetChoices_ReadsAListOfStrings(t *testing.T) {
	t.Parallel()

	choices := SubsetChoices(transferSchema(), map[string]any{
		"shipmentIds": []string{"shp_a", "shp_c"},
	})

	assert.Equal(t, []Choice{{ID: "shp_a", Label: "shp_a"}, {ID: "shp_c", Label: "shp_c"}},
		choices["shipmentIds"])
}

// A subset parameter the proposal left out, or set to something that is not
// a list, is still a subset: it lists no records rather than none at all.
func TestSubsetChoices_AnAbsentParameterListsNoRecords(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]map[string]any{
		"absent":     {},
		"null":       {"shipmentIds": nil},
		"not a list": {"shipmentIds": "shp_a"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			choices := SubsetChoices(transferSchema(), params)
			require.Contains(t, choices, "shipmentIds")
			assert.NotNil(t, choices["shipmentIds"])
			assert.Empty(t, choices["shipmentIds"])
		})
	}
}

// The list is bounded by the tool's own cap on the parameter, and never runs
// past MaxSubsetChoices whatever a schema declares.
func TestSubsetChoices_StopsAtTheParametersCap(t *testing.T) {
	t.Parallel()

	schema := transferSchema()
	property := schema[KeyProperties].(map[string]any)["shipmentIds"].(map[string]any)
	property[KeyMaxItems] = 2

	choices := SubsetChoices(schema, map[string]any{
		"shipmentIds": []any{"shp_a", "shp_b", "shp_c"},
	})
	assert.Equal(t, []Choice{{ID: "shp_a", Label: "shp_a"}, {ID: "shp_b", Label: "shp_b"}},
		choices["shipmentIds"])

	many := make([]any, 0, MaxSubsetChoices+1)
	for idx := range MaxSubsetChoices + 1 {
		many = append(many, "shp_"+strconv.Itoa(idx))
	}
	property[KeyMaxItems] = MaxSubsetChoices * 2
	assert.Len(t, SubsetChoices(schema, map[string]any{"shipmentIds": many})["shipmentIds"],
		MaxSubsetChoices)
}
