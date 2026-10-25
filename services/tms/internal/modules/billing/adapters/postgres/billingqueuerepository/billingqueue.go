package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var itemErrors = dberror.Mapping{Entity: "Billing queue item"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) billingqueueports.BillingQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
			bq := billingqueuedb.BillingQueueItems
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				bq.Where(keel.When(!req.IncludePosted, bq.Status.Ne(billingqueue.StatusPosted))),
				&listquery.Request[billingqueue.BillingQueueItem]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[billingqueue.BillingQueueItem],
					) *keel.Query[billingqueue.BillingQueueItem] {
						return q.Load(
							bq.Shipment().Customer(),
							bq.BillToCustomer(),
							bq.AssignedBiller(),
							bq.CanceledBy(),
						)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list billing queue items", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func itemDetail() *keel.Query[billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems
	ord := orderdb.Orders
	sh := bq.Shipment()
	po := keel.Scalar(ord.Pick(ord.PONumber).Where(ord.ID.EqCol(bq.OrderID)).Limit(1))

	return bq.Compute(bq.PONumber, keel.DynOf(po).Coalesce("")).
		Load(
			sh.Customer(),
			sh.AdditionalCharges(),
			sh.ChargeAllocations(),
			bq.BillToCustomer(),
			bq.AssignedBiller(),
			bq.CanceledBy(),
		)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *billingqueueports.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.BillingQueueItem, error) {
		entity, err := itemDetail().
			Where(billingqueuedb.BillingQueueItems.ID.Eq(req.ItemID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, itemErrors.Map(err, req.ItemID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.BillingQueueItem, error) {
		if err := billingqueuedb.BillingQueueItems.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create billing queue item", zap.Error(err))
			return nil, itemErrors.Map(err, entity.ID.String())
		}

		if err := r.syncShipmentTransferState(ctx, []pulid.ID{entity.ShipmentID}); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
			ItemID:     entity.ID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.BillingQueueItem, error) {
		bq := billingqueuedb.BillingQueueItems
		err := bq.Save(ctx, r.db.Keel(ctx), entity,
			bq.Status,
			bq.AssignedBillerID,
			bq.ExceptionReasonCode,
			bq.ReviewNotes,
			bq.ExceptionNotes,
			bq.ReviewStartedAt,
			bq.ReviewCompletedAt,
			bq.CanceledByID,
			bq.CanceledAt,
			bq.CancelReason,
			bq.HoldReasonCode,
			bq.HeldAt,
			bq.HeldByID,
			bq.StatusBeforeHold,
		)
		if err != nil {
			r.l.Error("failed to update billing queue item", zap.Error(err))
			return nil, itemErrors.Map(err, entity.ID.String())
		}

		if err = r.syncShipmentTransferState(ctx, []pulid.ID{entity.ShipmentID}); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
			ItemID:     entity.ID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) AttachInvoice(
	ctx context.Context,
	req *billingqueueports.AttachInvoiceRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req.InvoiceID.IsNil() || len(req.ItemIDs) == 0 {
			return 0, nil
		}

		bq := billingqueuedb.BillingQueueItems
		n, err := bq.Where(
			bq.ID.In(req.ItemIDs...),
			bq.Status.Eq(billingqueue.StatusApproved),
			bq.InvoiceID.IsNull(),
		).Update(ctx, r.db.Keel(ctx), bq.InvoiceID.To(req.InvoiceID))
		if err != nil {
			return 0, itemErrors.Map(err, req.InvoiceID.String())
		}

		return n, nil
	})
}

func (r *repository) MarkPostedForInvoice(
	ctx context.Context,
	req *billingqueueports.MarkPostedForInvoiceRequest,
) (int64, error) {
	if req.InvoiceID.IsNil() {
		return 0, nil
	}

	bq := billingqueuedb.BillingQueueItems
	affected, err := r.markPosted(ctx, bq.InvoiceID.Eq(req.InvoiceID))
	if err != nil {
		return 0, err
	}
	if affected > 0 || req.OrderID.IsNil() {
		return affected, nil
	}

	return r.markPosted(ctx, keel.And(
		bq.OrderID.Eq(req.OrderID),
		keel.When(len(req.ShipmentIDs) > 0, bq.ShipmentID.In(req.ShipmentIDs...)),
		keel.When(req.PayerID.IsNotNil(), bq.BillToCustomerID.Eq(req.PayerID)),
	))
}

func (r *repository) markPosted(ctx context.Context, scope keel.Pred) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		bq := billingqueuedb.BillingQueueItems
		rows, err := bq.Pick(bq.ShipmentID).
			Where(
				scope,
				bq.Status.Ne(billingqueue.StatusPosted),
				bq.Status.Ne(billingqueue.StatusCanceled),
			).
			UpdateReturning(ctx, r.db.Keel(ctx), bq.Status.To(billingqueue.StatusPosted))
		if err != nil {
			return 0, itemErrors.Map(err, "")
		}

		shipmentIDs := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			shipmentIDs = append(shipmentIDs, row.ShipmentID)
		}
		if err = r.syncShipmentTransferState(ctx, shipmentIDs); err != nil {
			return 0, err
		}

		return int64(len(rows)), nil
	})
}

func (r *repository) syncShipmentTransferState(ctx context.Context, shipmentIDs []pulid.ID) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return syncTransferState(ctx, r.db.Keel(ctx), shipmentIDs)
	})
}

func syncTransferState(ctx context.Context, db *keel.DB, shipmentIDs []pulid.ID) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	q, sets := shipmentTransferSync(shipmentIDs)
	_, err := q.Update(ctx, db, sets...)
	return err
}

func shipmentTransferSync(
	shipmentIDs []pulid.ID,
) (*keel.Query[shipment.Shipment], []keel.Assignment) {
	bq := billingqueuedb.BillingQueueItems
	sh := shipmentdb.Shipments

	latest := keel.CTE(
		"latest",
		[]string{"shipment_id", "status", "created_at"},
		bq.DistinctOn(bq.ShipmentID).
			Pick(bq.ShipmentID, bq.Status, bq.CreatedAt).
			Where(
				bq.ShipmentID.In(shipmentIDs...),
				bq.BillType.Eq(billingqueue.BillTypeInvoice),
			).
			Order(
				bq.ShipmentID.Asc(),
				keel.Asc(keel.Raw(transferStatusRankExpr, bq.Status)),
				bq.CreatedAt.Desc(),
				bq.ID.Desc(),
			),
	)

	q := sh.With(latest).
		Join(latest, latest.Col("shipment_id").EqCol(sh.ID)).
		Where(sh.ID.In(shipmentIDs...))

	return q, []keel.Assignment{
		sh.BillingTransferStatus.ToExpr(keel.DynOf(latest.Col("status")).Cast(keel.TypeString)),
		sh.TransferredToBillingAt.ToExpr(latest.Col("created_at")),
		sh.Version.ToExpr(sh.Version),
		sh.UpdatedAt.ToExpr(sh.UpdatedAt),
	}
}

func (r *repository) ReleaseForInvoice(
	ctx context.Context,
	req *billingqueueports.ReleaseForInvoiceRequest,
) ([]*billingqueue.BillingQueueItem, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueue.BillingQueueItem, error) {
			if req == nil || req.InvoiceID.IsNil() {
				return nil, nil
			}

			db := r.db.Keel(ctx)
			bq := billingqueuedb.BillingQueueItems
			items, err := bq.Where(
				keel.Or(
					bq.InvoiceID.Eq(req.InvoiceID),
					keel.When(req.AnchorItemID.IsNotNil(), bq.ID.Eq(req.AnchorItemID)),
				),
				bq.Status.Ne(billingqueue.StatusCanceled),
			).Lock(keel.ForUpdate).All(ctx, db)
			if err != nil {
				return nil, err
			}

			released := make([]*billingqueue.BillingQueueItem, 0, len(items))
			shipmentIDs := make([]pulid.ID, 0, len(items))
			for _, item := range items {
				if err = r.releaseItem(ctx, req, item); err != nil {
					return nil, err
				}
				released = append(released, item)
				if item.ShipmentID.IsNotNil() {
					shipmentIDs = append(shipmentIDs, item.ShipmentID)
				}
			}

			if err = syncTransferState(ctx, db, shipmentIDs); err != nil {
				return nil, err
			}

			return released, nil
		},
	)
}

func (r *repository) releaseItem(
	ctx context.Context,
	req *billingqueueports.ReleaseForInvoiceRequest,
	item *billingqueue.BillingQueueItem,
) error {
	target := billingqueue.StatusCanceled
	if req.Rebill {
		target = billingqueue.StatusApproved
	}
	if !billingqueue.IsAllowedVoidReleaseTransition(item.Status, target) {
		return errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Billing queue item {0} cannot be released from {1}",
			item.Number,
			string(item.Status),
		)
	}

	bq := billingqueuedb.BillingQueueItems
	sets := []keel.Assignment{bq.Status.To(target), bq.InvoiceID.ToNull()}
	item.Status = target
	item.InvoiceID = pulid.Nil

	if req.Rebill {
		if req.RenumberFn != nil {
			number, err := req.RenumberFn(ctx, item)
			if err != nil {
				return err
			}
			sets = append(sets, bq.Number.To(number))
			item.Number = number
		}
	} else {
		at := req.CanceledAt
		sets = append(sets,
			bq.CanceledByID.ToExpr(keel.Val(req.CanceledByID)),
			bq.CanceledAt.To(at),
			bq.CancelReason.To(req.CancelReason),
		)
		item.CanceledByID = req.CanceledByID
		item.CanceledAt = &at
		item.CancelReason = req.CancelReason
	}

	if _, err := bq.Where(bq.ID.Eq(item.ID)).Update(ctx, r.db.Keel(ctx), sets...); err != nil {
		return itemErrors.Map(err, item.ID.String())
	}
	item.Version++

	return nil
}

func (r *repository) ExistsByShipmentPayerAndType(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
	payerID pulid.ID,
	billType billingqueue.BillType,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		bq := billingqueuedb.BillingQueueItems
		return bq.Where(
			bq.ShipmentID.Eq(shipmentID),
			bq.BillToCustomerID.Eq(payerID),
			bq.BillType.Eq(billType),
			bq.Status.Ne(billingqueue.StatusCanceled),
		).Exists(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListActiveInvoiceItemsByShipmentIDs(
	ctx context.Context,
	req *billingqueueports.ListActiveInvoiceItemsRequest,
) (map[pulid.ID][]*billingqueue.BillingQueueItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*billingqueue.BillingQueueItem, error) {
			result := make(map[pulid.ID][]*billingqueue.BillingQueueItem, len(req.ShipmentIDs))
			if len(req.ShipmentIDs) == 0 {
				return result, nil
			}

			bq := billingqueuedb.BillingQueueItems
			items, err := bq.Where(
				bq.ShipmentID.In(req.ShipmentIDs...),
				bq.BillType.Eq(billingqueue.BillTypeInvoice),
				bq.Status.NotIn(billingqueue.StatusPosted, billingqueue.StatusCanceled),
			).Order(bq.ShipmentID.Asc(), bq.CreatedAt.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			for _, item := range items {
				result[item.ShipmentID] = append(result[item.ShipmentID], item)
			}

			return result, nil
		},
	)
}

func (r *repository) ListOpenShipmentIDs(
	ctx context.Context,
	req *billingqueueports.ListOpenShipmentIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req.Limit <= 0 {
			return []pulid.ID{}, nil
		}

		bq := billingqueuedb.BillingQueueItems
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), bq.
			Pick(bq.ShipmentID).
			Where(bq.Status.NotIn(billingqueue.StatusPosted, billingqueue.StatusCanceled)).
			Group(bq.ShipmentID).
			Order(bq.CreatedAt.Max().Desc()).
			Limit(req.Limit))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			return []pulid.ID{}, nil
		}

		return ids, nil
	})
}

type statusCount struct {
	Status billingqueue.Status
	Count  int
}

func (r *repository) GetStatusCounts(
	ctx context.Context,
	_ *billingqueueports.GetBillingQueueStatsRequest,
) (map[billingqueue.Status]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[billingqueue.Status]int, error) {
		bq := billingqueuedb.BillingQueueItems
		rows, err := keel.Into[statusCount](ctx, r.db.Keel(ctx), bq.
			Group(bq.Status).
			Pick(bq.Status, keel.Count()))
		if err != nil {
			return nil, err
		}

		counts := make(map[billingqueue.Status]int, len(rows))
		for _, row := range rows {
			counts[row.Status] = row.Count
		}

		return counts, nil
	})
}

func tenantInfo(entity *billingqueue.BillingQueueItem) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

const transferStatusRankExpr = `CASE ?
	WHEN 'SentBackToOps' THEN 0
	WHEN 'Exception' THEN 1
	WHEN 'OnHold' THEN 1
	WHEN 'ReadyForReview' THEN 2
	WHEN 'InReview' THEN 3
	WHEN 'Approved' THEN 5
	WHEN 'Canceled' THEN 8
	WHEN 'Posted' THEN 9
	ELSE 4 END`
