package receivablesql

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestOpenSoldWithoutRecourse(t *testing.T) {
	inv := invoicedb.Invoices
	keeltest.Snapshot(t, inv.Pick(inv.ID, OpenMinor(inv)).Where(Open(inv), SoldWithoutRecourse(inv)).Limit(1))
}
