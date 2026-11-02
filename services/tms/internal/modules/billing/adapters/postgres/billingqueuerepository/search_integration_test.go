//go:build integration

package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConnection_FindsAnItemByItsShipmentsProNumber(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	item := f.createItem(t, "BQ-SEARCH-1", billingqueue.StatusReadyForReview, billingqueue.BillTypeInvoice)
	var proNumber string
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		"SELECT pro_number FROM shipments WHERE id = $1", f.shipmentID,
	).Scan(&proNumber))
	require.NotEmpty(t, proNumber)

	search := func(query string) []pulid.ID {
		t.Helper()
		result, err := f.repo.ListConnection(f.ctx, &billingqueueports.ListBillingQueueItemsRequest{
			Filter: &pagination.QueryOptions{TenantInfo: f.tenant, Query: query},
			Cursor: pagination.CursorInfo{Limit: 50},
		})
		require.NoError(t, err)
		ids := make([]pulid.ID, 0, len(result.Items))
		for _, found := range result.Items {
			ids = append(ids, found.ID)
		}
		return ids
	}

	assert.Contains(t, search(proNumber), item.ID, "the PRO number the queue shows finds the item")
	assert.NotContains(t, search("NO-SUCH-PRO-404"), item.ID)
}

func TestListConnection_NarrowsToAShipmentsCustomerAndDeliveryDate(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	item := f.createItem(t, "BQ-FILTER-1", billingqueue.StatusReadyForReview, billingqueue.BillTypeInvoice)
	delivered := int64(1_793_000_000)
	var customerID pulid.ID
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`UPDATE shipments SET actual_delivery_date = $2 WHERE id = $1 RETURNING customer_id`,
		f.shipmentID, delivered,
	).Scan(&customerID))

	list := func(filters ...domaintypes.FieldFilter) []pulid.ID {
		t.Helper()
		result, err := f.repo.ListConnection(f.ctx, &billingqueueports.ListBillingQueueItemsRequest{
			Filter: &pagination.QueryOptions{TenantInfo: f.tenant, FieldFilters: filters},
			Cursor: pagination.CursorInfo{Limit: 200},
		})
		require.NoError(t, err)
		ids := make([]pulid.ID, 0, len(result.Items))
		for _, found := range result.Items {
			ids = append(ids, found.ID)
		}
		return ids
	}
	deliveredOn := func(from, to int64) domaintypes.FieldFilter {
		return domaintypes.FieldFilter{
			Field:    "shipment.actualDeliveryDate",
			Operator: dbtype.OpDateRange,
			Value:    map[string]any{"from": float64(from), "to": float64(to)},
		}
	}

	assert.Contains(t, list(
		domaintypes.FieldFilter{Field: "shipment.customerId", Operator: dbtype.OpEqual, Value: customerID.String()},
		deliveredOn(delivered-86_400, delivered),
	), item.ID, "its customer, delivered within the days asked for")
	assert.NotContains(t, list(deliveredOn(delivered+2*86_400, delivered+5*86_400)), item.ID,
		"delivered before the window")
	assert.NotContains(t, list(
		domaintypes.FieldFilter{Field: "shipment.customerId", Operator: dbtype.OpEqual, Value: "cus_not_this_one"},
	), item.ID)

	pickedUp := delivered - 2*86_400
	_, err := keel.Pool(f.db).Exec(f.ctx,
		`UPDATE shipments SET actual_ship_date = $2 WHERE id = $1`, f.shipmentID, pickedUp)
	require.NoError(t, err)
	pickedUpOn := func(from, to int64) domaintypes.FieldFilter {
		return domaintypes.FieldFilter{
			Field:    "shipment.actualShipDate",
			Operator: dbtype.OpDateRange,
			Value:    map[string]any{"from": float64(from), "to": float64(to)},
		}
	}
	assert.Contains(t, list(pickedUpOn(pickedUp-86_400, pickedUp+86_400)), item.ID,
		"the shipment's pickup reaches the item it is queued for")
	assert.NotContains(t, list(pickedUpOn(delivered, delivered+86_400)), item.ID)
}
