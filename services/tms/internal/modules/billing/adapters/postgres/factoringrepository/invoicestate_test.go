package factoringrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetedInvoices_PagesOneCustomerByIDWithoutAList(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, targetedInvoices(&factoringports.ListCustomerInvoiceIDsRequest{
		CustomerID:     pulid.ID("cus_1"),
		AfterInvoiceID: pulid.ID("inv_9"),
		Limit:          5000,
	}))

	assert.Contains(t, sql, `i."customer_id" = $`)
	assert.NotContains(t, sql, `ANY(`)
	assert.NotContains(t, sql, `"factoring_company_id" = $`)
	assert.Contains(t, sql, `i."id" > $`)
	assert.Contains(t, sql, `ORDER BY i."id" ASC LIMIT 1000`)
	assert.Contains(t, args, "cus_1")
	assert.Contains(t, args, "inv_9")
}

func TestTargetedInvoices_PagesOneFactor(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, targetedInvoices(&factoringports.ListCustomerInvoiceIDsRequest{
		FactoringCompanyID: pulid.ID("fc_1"),
	}))

	assert.Contains(t, sql, `i."factoring_company_id" = $`)
	assert.NotContains(t, sql, `"customer_id" = $`)
	assert.NotContains(t, sql, `i."id" > $`)
}

func TestCandidateInvoices_MergesDisjointPagesWithoutDeduplicating(t *testing.T) {
	t.Parallel()

	query, ok := candidateInvoices(&factoringports.ListCandidateInvoicesRequest{
		AfterInvoiceID: pulid.ID("inv_9"),
	}).(keeltest.Renderer)
	require.True(t, ok)
	sql, args := keeltest.Render(t, query)

	assert.Contains(t, sql, ` UNION ALL `)
	assert.NotContains(t, sql, ` UNION (`)
	assert.Contains(t, sql, `NOT EXISTS (SELECT 1 FROM "invoices" open_inv`)
	assert.Contains(t, sql, `open_inv."id" = fis."invoice_id"`)
	assert.Contains(t, sql, `open_inv."status" = 'Posted'`)
	assert.Contains(t, sql, `fis."verdict" <> 'Excluded'`)
	assert.Contains(t, sql, `ORDER BY c."id" ASC LIMIT 1000`)
	assert.Contains(t, args, "inv_9")
}
