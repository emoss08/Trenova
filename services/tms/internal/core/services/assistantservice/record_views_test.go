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

/*
A stop on the way is on the route.

SEED-PAY-003 picks up in Dallas, drops part of the load in Denver and
delivers the rest in Chicago. The card drew a Dallas–Chicago lane, as if
Denver had never happened.
*/
func TestShipmentView_DrawsTheStopsOnTheWay(t *testing.T) {
	t.Parallel()

	stop := func(kind, city, status, departed string) map[string]any {
		return map[string]any{
			"type": kind, "city": city, "state": "TX", "location": city + " Dock",
			"status": status, "actualDeparture": departed,
		}
	}
	view := recordView("shipment", map[string]any{
		"status": "InTransit",
		"moves": []any{map[string]any{"stops": []any{
			stop("Pickup", "Dallas", "Completed", "2026-09-21T23:00"),
			stop("SplitDelivery", "Denver", "Completed", "2026-09-22T05:00"),
			stop("Delivery", "Chicago", "New", ""),
		}}},
	})

	require.NotNil(t, view)
	assert.Equal(t, "Dallas, TX", view["from"].(map[string]any)["city"])
	assert.Equal(t, "Chicago, TX", view["to"].(map[string]any)["city"])
	via := view["via"].([]map[string]any)
	require.Len(t, via, 1)
	assert.Equal(t, "Denver, TX", via[0]["city"])
	assert.Equal(t, "SplitDelivery", via[0]["type"])
	assert.Equal(t, true, via[0]["done"])
	// Left Denver, the second of three evenly spaced stops, not yet in Chicago.
	assert.InDelta(t, 0.75, view["progress"], 0.001)
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

/*
The full record reads as well as the summary.

A delegate fetched SEED-PAY-007 with detail full, which nests each stop's
location, the driver and the equipment instead of writing them out, and the
card came out with an empty route and no driver, tractor or rate.
*/
func TestShipmentView_ReadsTheFullRecord(t *testing.T) {
	t.Parallel()

	stop := func(kind, city, state, name, arrived string) map[string]any {
		return map[string]any{
			"type": kind, "status": "Completed", "actualArrival": arrived,
			"actualDeparture": arrived,
			"location": map[string]any{
				"city": city, "name": name, "state": map[string]any{"abbreviation": state},
			},
		}
	}
	view := recordView("shipment", map[string]any{
		"status":            "ReadyToInvoice",
		"bol":               "BOL-2026-0107",
		"totalChargeAmount": "3050",
		"customer":          map[string]any{"name": "Range Logistics"},
		"serviceType":       map[string]any{"code": "STD"},
		"moves": []any{map[string]any{
			"loaded": true, "distance": float64(1016),
			"assignment": map[string]any{
				"primaryWorker": map[string]any{"firstName": "Emily", "lastName": "Chen", "wholeName": ""},
				"tractor":       map[string]any{"code": "TRC-003"},
				"trailer":       map[string]any{"code": "TRL-003"},
			},
			"stops": []any{
				stop("Pickup", "Denver", "CO", "Denver Drop Point", "2026-09-23 22:00 EDT (10 days ago)"),
				stop("Delivery", "Los Angeles", "CA", "Los Angeles Terminal", "2026-09-24 10:00 EDT (9 days ago)"),
			},
		}},
	})
	require.NotNil(t, view)
	from := view["from"].(map[string]any)
	to := view["to"].(map[string]any)
	assert.Equal(t, "Denver, CO", from["city"])
	assert.Equal(t, "Denver Drop Point", from["place"])
	assert.Equal(t, "Los Angeles, CA", to["city"])
	assert.Equal(t, "arrived", to["when"])

	got := map[string]any{}
	for _, entry := range view["facts"].([]any) {
		fact := entry.(map[string]any)
		got[fact["key"].(string)] = fact["value"]
	}
	assert.Equal(t, "Emily Chen", got["driver"])
	assert.Equal(t, "TRC-003", got["tractor"])
	assert.Equal(t, "TRL-003", got["trailer"])
	assert.Equal(t, "3050", got["rate"])
	assert.Equal(t, "STD", got["serviceType"])
}
