package assistantservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentView_IsTheRouteAndWhoIsMovingIt(t *testing.T) {
	t.Parallel()

	view := recordView("shipment", map[string]any{
		"id":       "shp_1",
		"status":   "InTransit",
		"bol":      "BOL-778",
		"customer": map[string]any{"id": "cus_1", "name": "Acme Manufacturing"},
		"rating":   map[string]any{"totalChargeAmount": "1486.00"},
		"commodities": []any{
			map[string]any{"weight": float64(38000)},
			map[string]any{"weight": float64(400)},
		},
		"moves": []any{map[string]any{
			"loaded":     true,
			"distance":   float64(356),
			"assignment": map[string]any{"primaryWorker": "Dana Ortiz", "tractor": "TRK-2214"},
			"stops": []any{
				map[string]any{
					"type": "Pickup", "status": "Completed", "city": "Chicago", "state": "IL",
					"location": "Acme DC 4", "actualDeparture": "2026-10-02T09:12",
				},
				map[string]any{
					"type": "Delivery", "status": "New", "city": "Columbus", "state": "OH",
					"location": "Acme Plant 2", "scheduledWindowStart": "2026-10-02T17:30",
				},
			},
		}},
	})

	require.NotNil(t, view)
	assert.Equal(t, "Acme Manufacturing", view["subtitle"])
	assert.Equal(t, map[string]any{
		"city": "Chicago, IL", "place": "Acme DC 4", "when": "departed", "at": "2026-10-02T09:12",
	}, view["from"])
	assert.Equal(t, "scheduled", view["to"].(map[string]any)["when"])
	assert.InDelta(t, 0.5, view["progress"], 0.001)
	assert.Equal(t, []any{
		map[string]any{"key": "driver", "value": "Dana Ortiz"},
		map[string]any{"key": "tractor", "value": "TRK-2214"},
		map[string]any{"key": "weight", "value": int64(38400)},
		map[string]any{"key": "miles", "value": float64(356)},
		map[string]any{"key": "rate", "value": "1486.00"},
		map[string]any{"key": "bol", "value": "BOL-778"},
	}, view["facts"])
}

func TestRecordView_OnlyForKindsThatHaveOne(t *testing.T) {
	t.Parallel()

	assert.Nil(t, recordView("customer", map[string]any{"id": "cus_1"}))
	invoice := recordView("invoice", map[string]any{
		"status": "Posted", "billToName": "FreshHaul Foods", "totalAmount": "2300.00", "currency": "USD",
	})
	assert.Equal(t, "FreshHaul Foods", invoice["subtitle"])
	assert.Equal(t, "2300.00", invoice["amount"].(map[string]any)["total"])
}
