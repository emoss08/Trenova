package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/shared/pulid"
)

func fillPONumber(ctx context.Context, db *keel.DB, itemID pulid.ID) error {
	bq := billingqueuedb.BillingQueueItems
	ord := orderdb.Orders
	po := keel.StringOf(keel.Scalar(ord.Pick(ord.PONumber.NullIf("")).Where(
		ord.ID.EqCol(bq.OrderID),
		ord.OrganizationID.EqCol(bq.OrganizationID),
		ord.BusinessUnitID.EqCol(bq.BusinessUnitID),
	).Limit(1)))

	_, err := bq.Where(bq.ID.Eq(itemID), bq.PONumber.IsNull(), bq.OrderID.IsNotNull()).
		Update(ctx, db, bq.PONumber.ToExpr(po), bq.Version.ToExpr(bq.Version))

	return err
}
