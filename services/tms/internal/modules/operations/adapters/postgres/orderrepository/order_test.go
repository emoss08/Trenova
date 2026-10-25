package orderrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestEmptyOrder_ExcludesEveryReferencedOrder(t *testing.T) {
	t.Parallel()

	repo := &repository{references: orderports.OrderReferences{
		invoicedb.Invoices.OrderID,
		billingqueuedb.BillingQueueItems.OrderID,
	}}
	keeltest.Snapshot(t, repo.emptyOrder(pulid.ID("ord_01")))
}
