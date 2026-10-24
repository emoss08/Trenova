package factoringsql

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

const itemJoin = "EXISTS (SELECT 1 FROM factoring_submission_items AS fsi WHERE " +
	"fsi.organization_id = inv.organization_id AND " +
	"fsi.business_unit_id = inv.business_unit_id AND " +
	"fsi.invoice_id = inv.id AND fsi.status IN "

func TestChargedBackInvoice(t *testing.T) {
	t.Parallel()

	assert.Equal(t, itemJoin+"('ChargedBack'))", ChargedBackInvoice())
}

func TestLiveInvoice(t *testing.T) {
	t.Parallel()

	assert.Equal(t, itemJoin+"('Pending', 'Funded'))", LiveInvoice())
}

func TestLiveInvoicePred_CorrelatesTheItemsWithTheInvoice(t *testing.T) {
	t.Parallel()

	inv := invoicedb.Invoices
	sql, args := keeltest.Render(t, inv.Pick(inv.ID).Where(LiveInvoicePred(inv)).Limit(1))

	assert.Contains(t, sql, `EXISTS (SELECT 1 FROM "factoring_submission_items" fsi WHERE `)
	assert.Contains(t, sql, `fsi."invoice_id" = i."id"`)
	assert.Contains(t, sql, `fsi."status" = ANY($`)
	assert.Contains(t, args, []string{"Pending", "Funded"})
}
