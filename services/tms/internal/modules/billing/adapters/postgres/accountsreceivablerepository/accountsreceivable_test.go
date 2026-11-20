package accountsreceivablerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestCustomerLedgerWindowNamesTheReplacedInvoiceEachRowBelongsTo(t *testing.T) {
	keeltest.Snapshot(t, ledgerWindowQuery(&customerledgerports.ListCustomerLedgerWindowRequest{
		CustomerID: pulid.ID("cus_1"),
		From:       100,
		Through:    200,
		Limit:      51,
	}))
}

func TestCustomerLedgerWindowPagesAfterAPosition(t *testing.T) {
	keeltest.Snapshot(t, ledgerWindowQuery(&customerledgerports.ListCustomerLedgerWindowRequest{
		CustomerID: pulid.ID("cus_1"),
		Through:    200,
		After: &customerledgerports.ARLedgerPosition{
			TransactionDate: 150,
			LineNumber:      2,
			ID:              pulid.ID("cle_1"),
		},
		Limit: 51,
	}))
}

func TestCustomerLedgerBalanceReadsClosedMonths(t *testing.T) {
	keeltest.Snapshot(t, closedMonthsQuery(pulid.ID("cus_1"), 1_767_225_600))
}

func TestCustomerLedgerBalanceReadsTheOpeningMonth(t *testing.T) {
	keeltest.Snapshot(t, openingMonthQuery(pulid.ID("cus_1"), 1_767_225_600, 1_768_435_200))
}
