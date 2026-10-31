package agenttoolports

import (
	"testing"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordIDParams_SendsAListToAToolThatTakesIDs(t *testing.T) {
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

	params, ok := RecordIDParams(schema, "inv_1")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"invoiceIds": []any{"inv_1"}}, params)
}

func TestRecordIDParams_SendsTheIDToAToolThatTakesOne(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipmentId": map[string]any{toolschema.KeyType: toolschema.TypeString},
		},
		toolschema.KeyRequired: []string{"shipmentId"},
	}

	params, ok := RecordIDParams(schema, "shp_1")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, params)
}

func TestRecordIDParams_FindsAListOfIDsBesideAnotherWayToName(t *testing.T) {
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

	params, ok := RecordIDParams(schema, "shp_1")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"shipmentIds": []any{"shp_1"}}, params)
}
