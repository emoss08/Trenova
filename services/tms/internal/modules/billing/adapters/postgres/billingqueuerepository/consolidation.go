package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

const defaultCandidateLimit = 5000

func endpointStop(
	sp *shipmentdb.ShipmentsTable,
	alias string,
	types []shipmentvocab.StopType,
	first bool,
) (*keel.Query[*shipment.Stop], keel.Expr) {
	stop := shipmentdb.Stops.As(alias)
	move := shipmentdb.ShipmentMoves.As(alias + "_move")
	order := []keel.Order{move.Sequence.Asc(), stop.Sequence.Asc()}
	if !first {
		order = []keel.Order{move.Sequence.Desc(), stop.Sequence.Desc()}
	}
	q := stop.
		Join(move, move.ID.EqCol(stop.ShipmentMoveID)).
		Pick(stop.LocationID).
		Where(move.ShipmentID.EqCol(sp.ID), stop.Type.In(types...)).
		Order(order...).
		Limit(1)
	return q, keel.CastText(stop.LocationID).Coalesce("")
}

func consolidationCandidates(
	req *billingqueueports.ListConsolidationCandidatesRequest,
	limit int,
) *keel.Query[*billingqueue.BillingQueueItem] {
	bqi := billingqueuedb.BillingQueueItems
	sp := shipmentdb.Shipments
	cus := customerdb.Customers
	cbp := customerdb.CustomerBillingProfiles
	ord := orderdb.Orders
	st := servicetypedb.ServiceTypes
	sib := shipmentdb.Shipments.As("sib")
	origin, originKey := endpointStop(
		sp,
		"origin",
		[]shipmentvocab.StopType{shipmentvocab.StopTypePickup, shipmentvocab.StopTypeSplitPickup},
		true,
	)
	dest, destKey := endpointStop(
		sp,
		"dest",
		[]shipmentvocab.StopType{
			shipmentvocab.StopTypeDelivery,
			shipmentvocab.StopTypeSplitDelivery,
		},
		false,
	)
	totalLegs := keel.CaseInt64().
		When(sp.OrderID.IsNull(), keel.Int64(0)).
		Else(keel.Int64Of(keel.Scalar(sib.
			Pick(keel.Count()).
			Where(sib.OrderID.EqCol(sp.OrderID), sib.Status.Ne(shipmentvocab.StatusCanceled)))))

	return bqi.
		Join(sp, sp.ID.EqCol(bqi.ShipmentID)).
		Join(cus, cus.ID.EqCol(bqi.BillToCustomerID)).
		LeftJoin(cbp, cbp.CustomerID.EqCol(cus.ID)).
		LeftJoin(ord, ord.ID.EqCol(sp.OrderID)).
		LeftJoin(st, st.ID.EqCol(sp.ServiceTypeID)).
		LeftJoinLateral(origin).
		LeftJoinLateral(dest).
		Pick(
			bqi.ID,
			bqi.ShipmentID,
			sp.OrderID.Coalesce(""),
			bqi.BillToCustomerID,
			cus.Name.Coalesce(""),
			sp.ProNumber.Coalesce(""),
			sp.BOL.Coalesce(""),
			ord.OrderNumber.Coalesce(""),
			ord.PONumber.Coalesce(""),
			st.Code.Coalesce(""),
			originKey,
			destKey,
			sp.ActualDeliveryDate,
			bqi.AllocatedTotalAmount,
			cbp.BillingCurrency.Coalesce("USD"),
			keel.Count().Over([]keel.Expr{sp.OrderID, bqi.BillToCustomerID}),
			totalLegs,
			cbp.SplitBy.Coalesce(customer.InvoiceSplitKeyCustomer),
			cbp.SectionBy.Coalesce(customer.InvoiceSectionKeyShipment),
			cbp.InvoiceDetail.Coalesce(customer.InvoiceDetailDetailed),
			cbp.MaxShipmentsPerInvoice.Coalesce(0),
			cbp.MinConsolidatedAmount,
			cbp.AutoBill.Coalesce(false),
		).
		Where(
			bqi.Status.Eq(billingqueue.StatusApproved),
			sp.Status.In(shipmentvocab.StatusReadyToInvoice, shipmentvocab.StatusCompleted),
			unbilledInvoiceFreight(bqi, sp, req.CustomerIDs, req.PeriodEnd),
		).
		Order(
			bqi.BillToCustomerID.Asc(),
			sp.ActualDeliveryDate.Asc().NullsLast(),
			sp.ProNumber.Asc(),
		).
		Limit(limit)
}

func (r *repository) ListConsolidationCandidates(
	ctx context.Context,
	req *billingqueueports.ListConsolidationCandidatesRequest,
) ([]*billingqueueports.ConsolidationCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.ConsolidationCandidate, error) {
			if len(req.CustomerIDs) == 0 {
				return nil, nil
			}

			limit := req.Limit
			if limit <= 0 {
				limit = defaultCandidateLimit
			}

			candidates, err := keel.Into[billingqueueports.ConsolidationCandidate](
				ctx,
				r.db.Keel(ctx),
				consolidationCandidates(req, limit),
			)
			if err != nil {
				r.l.Error("failed to list consolidation candidates", zap.Error(err))
				return nil, err
			}

			return sliceutils.Pointers(candidates), nil
		},
	)
}
