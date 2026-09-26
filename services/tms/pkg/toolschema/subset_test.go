package toolschema

import (
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
