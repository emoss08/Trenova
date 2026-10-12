package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil/mocks/glaccountportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// The person posting reads the entry by account, not by id: each journal line
// carries its account's code and name, alongside the shipment it bills.
func TestPreviewPostSummary_NamesTheAccountsTheEntryBooks(t *testing.T) {
	t.Parallel()

	f := newPostFixture(t)
	accounts := glaccountportsmock.NewMockGLAccountRepository(t)
	accounts.EXPECT().GetByIDs(mock.Anything, mock.MatchedBy(
		func(req glaccountports.GetGLAccountsByIDsRequest) bool {
			return assert.ElementsMatch(t, []pulid.ID{f.arID, f.revenueID}, req.GLAccountIDs) &&
				req.TenantInfo == f.tenant
		},
	)).Return([]*glaccount.GLAccount{
		{ID: f.arID, AccountCode: "1200", Name: "Accounts receivable"},
		{ID: f.revenueID, AccountCode: "4000", Name: "Freight revenue"},
	}, nil).Once()
	f.svc.glAccounts = accounts

	summary, err := f.svc.PreviewPostSummary(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	assert.Empty(t, summary.Refusals)
	assert.Equal(t, "INV-3001", summary.Number)
	assert.Equal(t, int64(125000), summary.TotalMinor)
	assert.Equal(t, []string{"PRO-88"}, summary.ShipmentsBilled)
	assert.True(t, summary.QueueItemPosted)
	require.NotNil(t, summary.Journal)
	assert.Equal(t, []invoiceports.InvoicePostJournalLine{
		{
			GLAccountID: f.arID,
			AccountCode: "1200",
			AccountName: "Accounts receivable",
			Description: summary.Journal.Lines[0].Description,
			DebitMinor:  125000,
		},
		{
			GLAccountID: f.revenueID,
			AccountCode: "4000",
			AccountName: "Freight revenue",
			Description: summary.Journal.Lines[1].Description,
			CreditMinor: 125000,
		},
	}, summary.Journal.Lines)
}

// A refused post is shown as its reasons and nothing else, so the dialog never
// offers to book an entry the post would not make.
func TestPreviewPostSummary_ShowsOnlyTheRefusalWhenPostWouldRefuse(t *testing.T) {
	t.Parallel()

	f := newPostFixture(t)
	f.invoice.Status = invoice.StatusVoided

	summary, err := f.svc.PreviewPostSummary(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	require.NotEmpty(t, summary.Refusals)
	assert.Contains(t, summary.Refusals[0], "Voided invoices cannot be posted")
	assert.Nil(t, summary.Journal)
	assert.Empty(t, summary.ShipmentsBilled)
	assert.Nil(t, summary.EDI)
}
