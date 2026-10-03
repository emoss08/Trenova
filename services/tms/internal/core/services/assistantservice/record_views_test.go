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

	assert.Nil(t, recordView("tractor", map[string]any{"id": "trc_1"}))
	invoice := recordView("invoice", map[string]any{
		"status": "Posted", "billToName": "FreshHaul Foods", "totalAmount": "2300.00", "currency": "USD",
	})
	assert.Equal(t, "FreshHaul Foods", invoice["subtitle"])
	assert.Equal(t, "2300.00", invoice["amount"].(map[string]any)["total"])
}

func TestCustomerView_IsHowTheyAreBilled(t *testing.T) {
	t.Parallel()

	view := recordView("customer", map[string]any{
		"id": "cus_1", "status": "Active", "code": "ACME", "city": "Chicago",
		"state": map[string]any{"abbreviation": "IL"},
		"billingProfile": map[string]any{
			"paymentTerm": "Net30", "creditStatus": "Active", "creditLimit": "50000.00",
			"creditBalance": "1250.00", "creditHoldReason": "",
		},
	})
	require.NotNil(t, view)
	assert.Equal(t, "ACME · Chicago, IL", view["subtitle"])
	keys := make([]string, 0)
	for _, entry := range view["facts"].([]any) {
		keys = append(keys, entry.(map[string]any)["key"].(string))
	}
	assert.Equal(t, []string{"paymentTerm", "creditStatus", "creditLimit", "creditBalance"}, keys)
}

func TestWorkerView_IsWhetherTheyCanTakeALoad(t *testing.T) {
	t.Parallel()

	view := recordView("worker", map[string]any{
		"id": "wrk_1", "status": "Active", "type": "Employee", "driverType": "OTR",
		"city": "Denver", "state": "CO", "canBeAssigned": false,
		"assignmentBlocked": "Medical card expired", "physicalDueDate": float64(1790000000),
		"mvrDueDate": "none on file",
	})
	require.NotNil(t, view)
	assert.Equal(t, "OTR · Employee · Denver, CO", view["subtitle"])
	ready := view["ready"].(map[string]any)
	assert.Equal(t, false, ready["canApprove"])
	assert.Equal(t, "Medical card expired", ready["blockedBy"])
	assert.Len(t, view["facts"], 2)
}
