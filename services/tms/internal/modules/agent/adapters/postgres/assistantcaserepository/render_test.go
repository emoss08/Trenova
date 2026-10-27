package assistantcaserepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDisputeRecordsQueryNamesTheInvoice(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, disputeRecordsQuery([]pulid.ID{"idsp_1"}))
	assert.Contains(t, sql, `LEFT JOIN "invoices"`)
}

func TestInvoiceQueries(t *testing.T) {
	t.Parallel()

	records, _ := keeltest.Render(t, invoiceRecordsQuery([]pulid.ID{"inv_1"}))
	assert.Contains(t, records, `"settlement_status"`)

	facts, _ := keeltest.Render(t, invoiceFactsQuery(pulid.ID("inv_1")))
	assert.Contains(t, facts, `"due_date"`)
}

func TestCarriersQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, carriersQuery([]pulid.ID{"car_1"}))
	assert.Contains(t, sql, `"name" ASC`)
}

func TestShipmentQueriesRender(t *testing.T) {
	t.Parallel()

	t.Run("records", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, shipmentRecordsQuery([]pulid.ID{"shp_1", "shp_2"}))
	})
	t.Run("facts", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, shipmentFactsQuery(&deskcaseports.GetShipmentCaseFactsRequest{
			ShipmentID: "shp_1",
			PODCode:    "POD",
		}))
	})
}
