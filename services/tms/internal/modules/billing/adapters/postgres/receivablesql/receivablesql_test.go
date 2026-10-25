package receivablesql

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestDump(t *testing.T) {
	inv := invoicedb.Invoices
	sql, args := keeltest.Render(t, inv.Pick(inv.ID, OpenMinor(inv)).Where(Open(inv), SoldWithoutRecourse(inv)).Limit(1))
	t.Log(sql)
	t.Log(args)
	_ = keel.Raw
}
