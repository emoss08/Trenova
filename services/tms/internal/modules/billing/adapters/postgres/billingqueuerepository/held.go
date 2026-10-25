package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

var heldStatuses = []billingqueue.Status{
	billingqueue.StatusReadyForReview,
	billingqueue.StatusInReview,
	billingqueue.StatusOnHold,
	billingqueue.StatusSentBackToOps,
	billingqueue.StatusException,
}

func unbilledInvoiceFreight(
	bqi *billingqueuedb.BillingQueueItemsTable,
	sp *shipmentdb.ShipmentsTable,
	customerIDs []pulid.ID,
	periodEnd int64,
) keel.Pred {
	line := invoicedb.InvoiceLines.As("il")
	inv := invoicedb.Invoices.As("inv")
	return keel.And(
		bqi.BillType.Eq(billingqueue.BillTypeInvoice),
		bqi.InvoiceID.IsNull(),
		bqi.IsAdjustmentOrigin.Eq(false),
		bqi.BillToCustomerID.In(customerIDs...),
		keel.NotExists(line.
			Join(inv, inv.ID.EqCol(line.InvoiceID)).
			Where(line.ShipmentID.EqCol(sp.ID), inv.CustomerID.EqCol(bqi.BillToCustomerID))),
		keel.DynOf(sp.ActualDeliveryDate).Coalesce(bqi.CreatedAt).Lt(periodEnd),
	)
}

func (r *repository) CountHeldForPeriod(
	ctx context.Context,
	req *billingqueueports.CountHeldForPeriodRequest,
) ([]*billingqueueports.HeldForPeriod, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.HeldForPeriod, error) {
			if len(req.CustomerIDs) == 0 {
				return nil, nil
			}

			bqi := billingqueuedb.BillingQueueItems
			sp := shipmentdb.Shipments
			held, err := keel.Into[billingqueueports.HeldForPeriod](ctx, r.db.Keel(ctx), bqi.
				Join(sp, sp.ID.EqCol(bqi.ShipmentID)).
				Pick(
					bqi.BillToCustomerID,
					keel.Count(),
					keel.DynOf(keel.Sum(bqi.AllocatedTotalAmount)).Coalesce(0),
				).
				Where(
					bqi.Status.In(heldStatuses...),
					sp.Status.Ne(shipment.StatusCanceled),
					unbilledInvoiceFreight(bqi, sp, req.CustomerIDs, req.PeriodEnd),
				).
				Group(bqi.BillToCustomerID))
			if err != nil {
				r.l.Error("failed to count held billing queue items", zap.Error(err))
				return nil, err
			}

			return sliceutils.Pointers(held), nil
		},
	)
}
