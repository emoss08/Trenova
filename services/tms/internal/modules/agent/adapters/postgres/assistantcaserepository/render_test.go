package assistantcaserepository

import (
	"testing"

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
