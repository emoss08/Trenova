package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordIDArgument_SendsAListToAToolThatTakesIDs(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"invoiceIds": map[string]any{
				toolschema.KeyType:  toolschema.TypeArray,
				toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
			},
		},
		toolschema.KeyRequired: []string{"invoiceIds"},
	}

	param, ok := recordIDParam(schema)
	require.True(t, ok)
	assert.Equal(t, "invoiceIds", param)
	assert.Equal(t, []any{"inv_1"}, recordIDArgument(schema, param, "inv_1"))
}

func TestRecordIDArgument_SendsTheIDToAToolThatTakesOne(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipmentId": map[string]any{toolschema.KeyType: toolschema.TypeString},
		},
		toolschema.KeyRequired: []string{"shipmentId"},
	}

	param, ok := recordIDParam(schema)
	require.True(t, ok)
	assert.Equal(t, "shp_1", recordIDArgument(schema, param, "shp_1"))
}

func TestRecordIDParam_FindsAListOfIDsBesideAnotherWayToName(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipmentIds": map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType:     toolschema.TypeString,
					toolschema.KeyRecordOf: "shipment",
				},
			},
			"proNumber": map[string]any{toolschema.KeyType: toolschema.TypeString},
		},
	}

	param, ok := recordIDParam(schema)
	require.True(t, ok)
	assert.Equal(t, "shipmentIds", param)
	assert.Equal(t, []any{"shp_1"}, recordIDArgument(schema, param, "shp_1"))
}
