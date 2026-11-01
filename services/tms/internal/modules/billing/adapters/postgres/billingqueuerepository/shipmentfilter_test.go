package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentFiltersCheckEachItemsShipmentByKey(t *testing.T) {
	t.Parallel()

	status := domaintypes.FieldFilter{Field: "status", Operator: dbtype.OpEqual, Value: "Approved"}
	original := &pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{
		status,
		{Field: "shipment.customerId", Operator: dbtype.OpEqual, Value: "cus_1"},
		{
			Field:    "shipment.actualDeliveryDate",
			Operator: dbtype.OpDateRange,
			Value:    map[string]any{"from": float64(100), "to": float64(200)},
		},
	}}

	q, filter := listQuery(&billingqueueports.ListBillingQueueItemsRequest{Filter: original})
	sql, args := keeltest.Render(t, listquery.Filter(q, filter))

	require.Len(t, original.FieldFilters, 3, "the caller's filter is left as it was")
	assert.Equal(t, []domaintypes.FieldFilter{status}, filter.FieldFilters)
	assert.Contains(t, sql, `(SELECT TRUE FROM "shipments" bqi_filter_shp WHERE `)
	assert.Contains(t, sql, `bqi_filter_shp."id" = bqi."shipment_id"`)
	assert.Contains(t, sql, `bqi_filter_shp."customer_id" = $`)
	assert.Contains(t, sql, `bqi_filter_shp."actual_delivery_date" >= $`)
	assert.Contains(t, sql, `) IS TRUE`)
	assert.NotContains(t, sql, `JOIN "shipments"`)
	assert.Contains(t, sql, `bqi."status" <> 'Posted'`)
	assert.Contains(t, args, "cus_1")
}

func TestShipmentFiltersLeaveOtherFiltersAlone(t *testing.T) {
	t.Parallel()

	original := &pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{
		{Field: "billToCustomerId", Operator: dbtype.OpEqual, Value: "cus_1"},
		{Field: "shipment.noSuchField", Operator: dbtype.OpEqual, Value: "x"},
		{Field: "shipment.customer.name", Operator: dbtype.OpEqual, Value: "x"},
	}}

	filter, onShipment := shipmentFilters(original)

	assert.Same(t, original, filter)
	assert.Nil(t, onShipment)

	none, pred := shipmentFilters(nil)
	assert.Nil(t, none)
	assert.Nil(t, pred)

	sql, _ := keeltest.Render(t, billingqueuedb.BillingQueueItems.Where(onShipment))
	assert.NotContains(t, sql, "bqi_filter_shp")
}
