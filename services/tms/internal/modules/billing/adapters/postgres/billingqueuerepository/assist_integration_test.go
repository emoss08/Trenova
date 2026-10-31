//go:build integration

package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecentCustomerInvoices_ReadsChargesWithoutACode(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	var customerID pulid.ID
	require.NoError(t, f.db.Pool().QueryRow(f.ctx,
		`SELECT i.customer_id FROM invoices i
		JOIN invoice_lines il ON il.invoice_id = i.id
		WHERE i.organization_id = $1 AND i.business_unit_id = $2
			AND i.status = 'Posted' AND i.bill_type = 'Invoice'
			AND i.is_adjustment_artifact IS FALSE AND il.charge_code IS NULL
		ORDER BY i.invoice_date DESC, i.id DESC LIMIT 1`,
		f.tenant.OrgID, f.tenant.BuID,
	).Scan(&customerID))

	recent, err := f.repo.RecentCustomerInvoices(f.ctx, f.tenant, customerID, 1)
	require.NoError(t, err, "a line with no charge code is read, not a scan error")
	require.Len(t, recent, 1)

	uncoded := 0
	for _, charge := range recent[0].Charges {
		if charge.Code == "" {
			uncoded++
			assert.Contains(t,
				[]invoice.InvoiceLineType{invoice.InvoiceLineTypeFreight, invoice.InvoiceLineTypeMemo}, charge.Type)
		}
	}
	assert.Positive(t, uncoded)
}
