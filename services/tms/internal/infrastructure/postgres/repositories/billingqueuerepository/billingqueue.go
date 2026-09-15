package billingqueuerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BillingQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListBillingQueueItemsRequest,
) *bun.SelectQuery {
	bqi := buncolgen.BillingQueueItemColumns

	q = querybuilder.ApplyFilters(
		q,
		buncolgen.BillingQueueItemTable.Alias,
		req.Filter,
		(*billingqueue.BillingQueueItem)(nil),
	)

	if !req.IncludePosted {
		q = q.Where(bqi.Status.Ne(), billingqueue.StatusPosted)
	}

	q = q.Relation(buncolgen.BillingQueueItemRelations.Shipment).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.Customer)).
		Relation(buncolgen.BillingQueueItemRelations.BillToCustomer).
		Relation(buncolgen.BillingQueueItemRelations.AssignedBiller).
		Relation(buncolgen.BillingQueueItemRelations.CanceledBy)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*billingqueue.BillingQueueItem, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count billing queue items", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*billingqueue.BillingQueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)
	entity := new(billingqueue.BillingQueueItem)

	if err := db.NewSelect().
		Model(entity).
		Where(bqi.ID.Eq(), req.ItemID).
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		Relation(buncolgen.BillingQueueItemRelations.Shipment).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.Customer)).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.AdditionalCharges)).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.ChargeAllocations)).
		Relation(buncolgen.BillingQueueItemRelations.BillToCustomer).
		Relation(buncolgen.BillingQueueItemRelations.AssignedBiller).
		Relation(buncolgen.BillingQueueItemRelations.CanceledBy).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Billing queue item")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert billing queue item: %w", err)
	}

	if err := r.syncShipmentTransferState(
		ctx,
		tenantInfo(entity),
		[]pulid.ID{entity.ShipmentID},
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	result, err := db.NewUpdate().
		Model(entity).
		Where(bqi.ID.Eq(), entity.ID).
		Where(bqi.OrganizationID.Eq(), entity.OrganizationID).
		Where(bqi.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(bqi.Version.Eq(), entity.Version).
		Set(bqi.Status.Set(), entity.Status).
		Set(bqi.AssignedBillerID.Set(), entity.AssignedBillerID).
		Set(bqi.ExceptionReasonCode.Set(), entity.ExceptionReasonCode).
		Set(bqi.ReviewNotes.Set(), entity.ReviewNotes).
		Set(bqi.ExceptionNotes.Set(), entity.ExceptionNotes).
		Set(bqi.ReviewStartedAt.Set(), entity.ReviewStartedAt).
		Set(bqi.ReviewCompletedAt.Set(), entity.ReviewCompletedAt).
		Set(bqi.CanceledByID.Set(), entity.CanceledByID).
		Set(bqi.CanceledAt.Set(), entity.CanceledAt).
		Set(bqi.CancelReason.Set(), entity.CancelReason).
		Set(bqi.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update billing queue item: %w", err)
	}

	if rowsErr := dberror.CheckRowsAffected(
		result,
		"Billing queue item",
		entity.ID.String(),
	); rowsErr != nil {
		return nil, rowsErr
	}

	if err = r.syncShipmentTransferState(
		ctx,
		tenantInfo(entity),
		[]pulid.ID{entity.ShipmentID},
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

// AttachInvoice links the given billing-queue items to the invoice that bills them.
// Only items still approved and not on an invoice are linked, and the count says
// how many were, so a caller can detect an item billed elsewhere in the meantime.
func (r *repository) AttachInvoice(
	ctx context.Context,
	req *repositories.AttachInvoiceRequest,
) (int64, error) {
	if req.InvoiceID.IsNil() || len(req.ItemIDs) == 0 {
		return 0, nil
	}

	bqi := buncolgen.BillingQueueItemColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*billingqueue.BillingQueueItem)(nil)).
		Where(bqi.ID.In(), bun.List(req.ItemIDs)).
		Where(bqi.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(bqi.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(bqi.Status.Eq(), billingqueue.StatusApproved).
		Where(bqi.InvoiceID.IsNull()).
		Set(bqi.InvoiceID.Set(), req.InvoiceID).
		Set(bqi.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("attach invoice to billing queue items: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return affected, nil
}

// MarkPostedForInvoice flips every non-terminal billing-queue item an invoice bills to
// Posted, so a grouped or consolidated invoice settles all of its items and not just
// the anchor. The invoice back-link is the predicate; the order fallback exists only
// for rows written before that link, which the migration could not always resolve.
func (r *repository) MarkPostedForInvoice(
	ctx context.Context,
	req *repositories.MarkPostedForInvoiceRequest,
) (int64, error) {
	if req.InvoiceID.IsNil() {
		return 0, nil
	}

	bqi := buncolgen.BillingQueueItemColumns
	affected, err := r.markPosted(ctx, req.TenantInfo, func(q *bun.UpdateQuery) *bun.UpdateQuery {
		return q.Where(bqi.InvoiceID.Eq(), req.InvoiceID)
	})
	if err != nil {
		return 0, err
	}
	if affected > 0 || req.OrderID.IsNil() {
		return affected, nil
	}

	return r.markPosted(ctx, req.TenantInfo, func(q *bun.UpdateQuery) *bun.UpdateQuery {
		q = q.Where(bqi.OrderID.Eq(), req.OrderID)
		if len(req.ShipmentIDs) > 0 {
			q = q.Where(bqi.ShipmentID.In(), bun.List(req.ShipmentIDs))
		}
		if req.PayerID.IsNotNil() {
			q = q.Where(bqi.BillToCustomerID.Eq(), req.PayerID)
		}
		return q
	})
}

func (r *repository) markPosted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	scope func(*bun.UpdateQuery) *bun.UpdateQuery,
) (int64, error) {
	bqi := buncolgen.BillingQueueItemColumns
	q := r.db.DBForContext(ctx).NewUpdate().
		Model((*billingqueue.BillingQueueItem)(nil)).
		Where(bqi.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(bqi.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(bqi.Status.Ne(), billingqueue.StatusPosted).
		Where(bqi.Status.Ne(), billingqueue.StatusCanceled).
		Set(bqi.Status.Set(), billingqueue.StatusPosted).
		Set(bqi.Version.Inc(1)).
		Returning(bqi.ShipmentID.Bare())

	shipmentIDs := make([]pulid.ID, 0)
	if _, err := scope(q).Exec(ctx, &shipmentIDs); err != nil {
		return 0, fmt.Errorf("mark billing queue items posted: %w", err)
	}

	if err := r.syncShipmentTransferState(ctx, tenantInfo, shipmentIDs); err != nil {
		return 0, err
	}

	return int64(len(shipmentIDs)), nil
}

// syncShipmentTransferState copies each shipment's billing-queue state onto the
// shipment inside the caller's transaction. Every write to a billing queue item
// goes through this repository, so this is the only writer of the shipment's
// billing_transfer_status and transferred_to_billing_at columns.
//
// A split-billed shipment carries one invoice item per payer, and the shipment
// reflects the least advanced of them: it is not posted until every payer's item
// is, and a payer's item still in review keeps it in review.
func (r *repository) syncShipmentTransferState(
	ctx context.Context,
	ti pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) error {
	if len(shipmentIDs) == 0 {
		return nil
	}

	bqi := buncolgen.BillingQueueItemColumns
	sp := buncolgen.ShipmentColumns
	db := r.db.DBForContext(ctx)

	latest := db.NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		DistinctOn(bqi.ShipmentID.Qualified()).
		Column(bqi.ShipmentID.Bare(), bqi.Status.Bare(), bqi.CreatedAt.Bare()).
		Apply(buncolgen.BillingQueueItemApplyTenant(ti)).
		Where(bqi.ShipmentID.In(), bun.List(shipmentIDs)).
		Where(bqi.BillType.Eq(), billingqueue.BillTypeInvoice).
		OrderExpr(bqi.ShipmentID.OrderAsc()).
		OrderExpr(bqi.Status.Expr(transferStatusRankExpr)+" ASC").
		Order(bqi.CreatedAt.OrderDesc(), bqi.ID.OrderDesc())

	if _, err := db.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		TableExpr("(?) AS latest", latest).
		Set(sp.BillingTransferStatus.SetExpr("latest.status::text")).
		Set(sp.TransferredToBillingAt.SetExpr("latest.created_at")).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentScopeTenantUpdate(uq, ti).
				Where(sp.ID.Expr("{} = latest.shipment_id"))
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("sync shipment billing transfer state: %w", err)
	}

	return nil
}

// ReleaseForInvoice hands back every item a voided invoice billed. The anchor
// is found by id because a draft never wrote the back-link; the rest by the
// link. Released items get a fresh number: the voided invoice keeps the old one.
func (r *repository) ReleaseForInvoice(
	ctx context.Context,
	req *repositories.ReleaseForInvoiceRequest,
) ([]*billingqueue.BillingQueueItem, error) {
	if req == nil || req.InvoiceID.IsNil() {
		return nil, nil
	}

	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	items := make([]*billingqueue.BillingQueueItem, 0, 1)
	if err := db.NewSelect().
		Model(&items).
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereGroup(" OR ", func(inner *bun.SelectQuery) *bun.SelectQuery {
				inner = inner.Where(bqi.InvoiceID.Eq(), req.InvoiceID)
				if req.AnchorItemID.IsNotNil() {
					inner = inner.WhereOr(bqi.ID.Eq(), req.AnchorItemID)
				}
				return inner
			})
		}).
		Where(bqi.Status.Ne(), billingqueue.StatusCanceled).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load billing queue items for release: %w", err)
	}

	released := make([]*billingqueue.BillingQueueItem, 0, len(items))
	shipmentIDs := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		target := billingqueue.StatusCanceled
		if req.Rebill {
			target = billingqueue.StatusApproved
		}
		if !billingqueue.IsAllowedVoidReleaseTransition(item.Status, target) {
			return nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Billing queue item {0} cannot be released from {1}",
				item.Number,
				string(item.Status),
			)
		}

		update := db.NewUpdate().
			Model((*billingqueue.BillingQueueItem)(nil)).
			Where(bqi.ID.Eq(), item.ID).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.BillingQueueItemScopeTenantUpdate(uq, req.TenantInfo)
			}).
			Set(bqi.Status.Set(), target).
			Set(bqi.InvoiceID.SetNull()).
			Set(bqi.Version.Inc(1))
		item.Status = target
		item.InvoiceID = pulid.Nil

		if req.Rebill {
			if req.RenumberFn != nil {
				number, err := req.RenumberFn(ctx, item.BillType)
				if err != nil {
					return nil, err
				}
				update = update.Set(bqi.Number.Set(), number)
				item.Number = number
			}
		} else {
			update = update.
				Set(bqi.CanceledByID.Set(), req.CanceledByID).
				Set(bqi.CanceledAt.Set(), req.CanceledAt).
				Set(bqi.CancelReason.Set(), req.CancelReason)
			item.CanceledByID = req.CanceledByID
			at := req.CanceledAt
			item.CanceledAt = &at
			item.CancelReason = req.CancelReason
		}
		if _, err := update.Exec(ctx); err != nil {
			return nil, fmt.Errorf("release billing queue item %s: %w", item.ID, err)
		}
		released = append(released, item)
		if item.ShipmentID.IsNotNil() {
			shipmentIDs = append(shipmentIDs, item.ShipmentID)
		}
	}

	if err := r.syncShipmentTransferState(ctx, req.TenantInfo, shipmentIDs); err != nil {
		return nil, err
	}

	return released, nil
}

func (r *repository) ExistsByShipmentPayerAndType(
	ctx context.Context,
	ti pagination.TenantInfo,
	shipmentID pulid.ID,
	payerID pulid.ID,
	billType billingqueue.BillType,
) (bool, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	return db.NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		Where(bqi.ShipmentID.Eq(), shipmentID).
		Where(bqi.BillToCustomerID.Eq(), payerID).
		Where(bqi.BillType.Eq(), billType).
		Where(bqi.Status.Ne(), billingqueue.StatusCanceled).
		Apply(buncolgen.BillingQueueItemApplyTenant(ti)).
		Exists(ctx)
}

// ListActiveInvoiceItemsByShipmentIDs returns, per shipment, the invoice items
// that are neither posted nor canceled. Posting reads it to decide whether the
// shipment is fully invoiced or another payer's share is still open.
func (r *repository) ListActiveInvoiceItemsByShipmentIDs(
	ctx context.Context,
	req *repositories.ListActiveInvoiceItemsRequest,
) (map[pulid.ID][]*billingqueue.BillingQueueItem, error) {
	result := make(map[pulid.ID][]*billingqueue.BillingQueueItem, len(req.ShipmentIDs))
	if len(req.ShipmentIDs) == 0 {
		return result, nil
	}

	bqi := buncolgen.BillingQueueItemColumns
	items := make([]*billingqueue.BillingQueueItem, 0, len(req.ShipmentIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&items).
		Where(bqi.ShipmentID.In(), bun.List(req.ShipmentIDs)).
		Where(bqi.BillType.Eq(), billingqueue.BillTypeInvoice).
		Where(bqi.Status.NotIn(), bun.List([]billingqueue.Status{
			billingqueue.StatusPosted,
			billingqueue.StatusCanceled,
		})).
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		Order(bqi.ShipmentID.OrderAsc(), bqi.CreatedAt.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list active billing queue items: %w", err)
	}

	for _, item := range items {
		result[item.ShipmentID] = append(result[item.ShipmentID], item)
	}

	return result, nil
}

func (r *repository) GetStatusCounts(
	ctx context.Context,
	req *repositories.GetBillingQueueStatsRequest,
) (map[billingqueue.Status]int, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	var rows []struct {
		Status billingqueue.Status `bun:"status"`
		Count  int                 `bun:"count"`
	}

	if err := db.NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		ColumnExpr(bqi.Status.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		GroupExpr(bqi.Status.Qualified()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	counts := make(map[billingqueue.Status]int, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}

	return counts, nil
}

func tenantInfo(entity *billingqueue.BillingQueueItem) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

// transferStatusRankExpr orders billing-queue statuses from least to most
// advanced, so the shipment's transfer state follows the payer furthest behind.
const transferStatusRankExpr = `CASE {}
	WHEN 'SentBackToOps' THEN 0
	WHEN 'Exception' THEN 1
	WHEN 'OnHold' THEN 1
	WHEN 'ReadyForReview' THEN 2
	WHEN 'InReview' THEN 3
	WHEN 'Approved' THEN 5
	WHEN 'Canceled' THEN 8
	WHEN 'Posted' THEN 9
	ELSE 4 END`
