package billingqueuerepository

import (
	"context"
	"math"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

const secondsPerDay = 86400

var decimalSecondsPerDay = decimal.NewFromInt(secondsPerDay)

type laneStatRow struct {
	Lane       billingqueue.Lane
	Count      int
	TotalMinor int64
}

func (r *repository) GetLaneStats(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*billingqueueports.BillingQueueLaneStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.BillingQueueLaneStat, error) {
			rows, err := keel.Into[laneStatRow](ctx, r.db.Keel(ctx), laneStats())
			if err != nil {
				return nil, err
			}

			out := make([]*billingqueueports.BillingQueueLaneStat, 0, len(rows))
			for _, row := range rows {
				out = append(out, &billingqueueports.BillingQueueLaneStat{
					Lane:  row.Lane,
					Count: row.Count,
					Total: money.DecimalFromMinor(row.TotalMinor),
				})
			}

			return out, nil
		},
	)
}

func laneStats() keel.Query[*billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems

	return bq.Pick(bq.Lane, keel.Count(), bq.AllocatedTotalAmountMinor.Sum()).
		Where(bq.Lane.Ne(billingqueue.LaneClosed)).
		Group(bq.Lane)
}

type approvedRow struct {
	Count         int
	TotalMinor    *int64
	MedianSeconds *float64
}

type lagRow struct {
	Current decimal.NullDecimal
	Prior   decimal.NullDecimal
}

func (r *repository) GetPulse(
	ctx context.Context,
	req *billingqueueports.GetBillingQueuePulseRequest,
) (*billingqueueports.BillingQueuePulse, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueueports.BillingQueuePulse, error) {
			db := r.db.Keel(ctx)

			oldest, err := keel.Into[*int64](ctx, db, oldestWaiting())
			if err != nil {
				return nil, err
			}
			approved, err := keel.Into[approvedRow](ctx, db, approvedSince(req.DayStart))
			if err != nil {
				return nil, err
			}
			lags, err := keel.Into[lagRow](
				ctx,
				db,
				deliveryToInvoice(req.WindowStart, req.PriorStart),
			)
			if err != nil {
				return nil, err
			}

			pulse := &billingqueueports.BillingQueuePulse{ApprovedTodayTotal: decimal.Zero}
			if len(oldest) > 0 {
				pulse.OldestWaitingAt = oldest[0]
			}
			if len(approved) > 0 {
				row := approved[0]
				pulse.ApprovedTodayCount = row.Count
				if row.TotalMinor != nil {
					pulse.ApprovedTodayTotal = money.DecimalFromMinor(*row.TotalMinor)
				}
				if row.MedianSeconds != nil {
					median := int64(math.Round(*row.MedianSeconds))
					pulse.MedianReviewSecondsToday = &median
				}
			}
			if len(lags) > 0 {
				pulse.DeliveryToInvoiceDays = inDays(lags[0].Current)
				pulse.PriorDeliveryToInvoice = inDays(lags[0].Prior)
			}

			return pulse, nil
		},
	)
}

func oldestWaiting() keel.Query[*billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems

	return bq.Pick(bq.CreatedAt.Min()).Where(bq.Lane.In(billingqueue.ReviewLanes()...))
}

func approvedSince(dayStart int64) keel.Query[*billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems
	duration := bq.ReviewCompletedAt.Sub(bq.ReviewStartedAt)

	return bq.Pick(
		keel.Count(),
		bq.AllocatedTotalAmountMinor.Sum(),
		keel.PercentileCont(0.5, duration.Asc()).Filter(bq.ReviewStartedAt.IsNotNull()),
	).Where(
		bq.ReviewCompletedAt.Gte(dayStart),
		bq.Status.In(billingqueue.StatusApproved, billingqueue.StatusPosted),
	)
}

func deliveryToInvoice(windowStart, priorStart int64) keel.Query[*invoice.Invoice] {
	inv := invoicedb.Invoices
	sp := shipmentdb.Shipments
	lag := inv.CreatedAt.Sub(sp.LastDeliveryArrivalAt)

	return inv.Pick(
		lag.Avg().Filter(inv.CreatedAt.Gte(windowStart)),
		lag.Avg().Filter(inv.CreatedAt.Lt(windowStart)),
	).Join(
		sp,
		sp.ID.EqCol(inv.ShipmentID),
		sp.OrganizationID.EqCol(inv.OrganizationID),
		sp.BusinessUnitID.EqCol(inv.BusinessUnitID),
	).Where(
		inv.CreatedAt.Gte(priorStart),
		inv.BillType.Eq(billingqueue.BillTypeInvoice),
		inv.IsAdjustmentArtifact.IsFalse(),
		inv.Status.Ne(invoice.StatusVoided),
		sp.LastDeliveryArrivalAt.IsNotNull(),
	)
}

func inDays(seconds decimal.NullDecimal) decimal.NullDecimal {
	if !seconds.Valid {
		return seconds
	}

	return decimal.NewNullDecimal(seconds.Decimal.Div(decimalSecondsPerDay).Round(1))
}

func laneFilter(req *billingqueueports.ListBillingQueueItemsRequest) keel.Pred {
	bq := billingqueuedb.BillingQueueItems

	return keel.And(
		keel.When(len(req.Lanes) > 0, bq.Lane.In(req.Lanes...)),
		keel.When(
			req.MinAmount.Valid,
			bq.AllocatedTotalAmountMinor.Gte(money.MinorUnits(req.MinAmount.Decimal)),
		),
		keel.When(
			req.MaxAmount.Valid,
			bq.AllocatedTotalAmountMinor.Lte(money.MinorUnits(req.MaxAmount.Decimal)),
		),
		keel.When(req.CreatedBefore != nil, bq.CreatedAt.Lt(derefInt64(req.CreatedBefore))),
	)
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}

	return *v
}
