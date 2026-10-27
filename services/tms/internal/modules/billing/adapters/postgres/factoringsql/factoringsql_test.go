package factoringsql

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestLiveInvoicePred_CorrelatesTheItemsWithTheInvoice(t *testing.T) {
	t.Parallel()

	inv := invoicedb.Invoices
	sql, args := keeltest.Render(t, inv.Pick(inv.ID).Where(LiveInvoicePred(inv)).Limit(1))

	assert.Contains(t, sql, `EXISTS (SELECT 1 FROM "factoring_submission_items" fsi WHERE `)
	assert.Contains(t, sql, `fsi."invoice_id" = i."id"`)
	assert.Contains(t, sql, `fsi."status" IN ('Pending', 'Funded')`)
	assert.Equal(t, []any{"org_1", "bu_1"}, args)
}
