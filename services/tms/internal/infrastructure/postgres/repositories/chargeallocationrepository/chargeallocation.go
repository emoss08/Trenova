package chargeallocationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.ChargeAllocationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.charge-allocation-repository"),
	}
}

func (r *repository) SyncForShipment(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	if entity == nil {
		return nil
	}
	ti := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	if entity.ChargeAllocations != nil {
		if err := r.syncShipmentRows(ctx, tx, entity); err != nil {
			return err
		}
	}

	if err := r.followCustomerChange(ctx, tx, entity); err != nil {
		return err
	}

	if err := r.dropOrphans(ctx, tx, entity); err != nil {
		return err
	}

	return r.RefreshQueueSnapshots(ctx, tx, ti, entity.ID)
}

func (r *repository) syncShipmentRows(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	cols := buncolgen.ChargeAllocationColumns
	ti := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	existing, err := r.existingForShipment(ctx, tx, ti, entity.ID)
	if err != nil {
		return err
	}

	kept := make(map[pulid.ID]struct{}, len(entity.ChargeAllocations))
	for _, allocation := range entity.ChargeAllocations {
		if allocation == nil || allocation.ChargeKind == shipment.ChargeAllocationKindOrderCharge {
			continue
		}
		if err = r.normalize(entity, allocation); err != nil {
			return err
		}

		switch {
		case allocation.ID.IsNil():
			allocation.ID = pulid.MustNew("chal_")
			if err = r.insert(ctx, tx, allocation); err != nil {
				return err
			}
		case existing[allocation.ID] != nil:
			if err = r.update(ctx, tx, allocation, existing[allocation.ID]); err != nil {
				return err
			}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown charge allocation").
				WithParam("chargeAllocationId", allocation.ID.String())
		}
		kept[allocation.ID] = struct{}{}
	}

	deleteIDs := make([]pulid.ID, 0, len(existing))
	for id, row := range existing {
		if _, ok := kept[id]; ok {
			continue
		}
		if row.IsInvoiced() {
			return errortypes.NewValidationError(
				"chargeAllocations",
				errortypes.ErrInvalidOperation,
				"An allocation already carried on an invoice cannot be removed",
			)
		}
		deleteIDs = append(deleteIDs, id)
	}
	if len(deleteIDs) == 0 {
		return nil
	}

	if _, err = tx.NewDelete().
		Model((*shipment.ChargeAllocation)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ChargeAllocationScopeTenantDelete(dq, ti).
				Where(cols.ID.In(), bun.List(deleteIDs))
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete charge allocations: %w", err)
	}

	return nil
}

// normalize seats tenancy and the charge target on a row. A row that arrived
// pointing at a charge by position takes that charge's freshly minted id, which
// is what lets a new accessorial and its split be saved together.
func (r *repository) normalize(entity *shipment.Shipment, allocation *shipment.ChargeAllocation) error {
	allocation.OrganizationID = entity.OrganizationID
	allocation.BusinessUnitID = entity.BusinessUnitID
	shipmentID := entity.ID
	allocation.ShipmentID = &shipmentID
	allocation.OrderChargeID = nil

	switch allocation.ChargeKind {
	case shipment.ChargeAllocationKindFreight:
		allocation.AdditionalChargeID = nil
		allocation.AdditionalChargeIndex = nil
	case shipment.ChargeAllocationKindAccessorial:
		if allocation.AdditionalChargeIndex != nil {
			idx := *allocation.AdditionalChargeIndex
			if idx < 0 || idx >= len(entity.AdditionalCharges) || entity.AdditionalCharges[idx] == nil ||
				entity.AdditionalCharges[idx].ID.IsNil() {
				return errortypes.NewValidationError(
					"chargeAllocations",
					errortypes.ErrInvalid,
					"Allocation points at an additional charge that does not exist",
				)
			}
			id := entity.AdditionalCharges[idx].ID
			allocation.AdditionalChargeID = &id
			allocation.AdditionalChargeIndex = nil
		}
		if allocation.AdditionalChargeID == nil || allocation.AdditionalChargeID.IsNil() {
			return errortypes.NewValidationError(
				"chargeAllocations",
				errortypes.ErrRequired,
				"An accessorial allocation must name its charge",
			)
		}
		found := false
		for _, charge := range entity.AdditionalCharges {
			if charge != nil && charge.ID == *allocation.AdditionalChargeID {
				found = true
				break
			}
		}
		if !found && entity.AdditionalCharges != nil {
			return errortypes.NewValidationError(
				"chargeAllocations",
				errortypes.ErrInvalid,
				"Allocation points at an additional charge that is not on this shipment",
			)
		}
	case shipment.ChargeAllocationKindOrderCharge:
	}

	return nil
}

func (r *repository) insert(ctx context.Context, tx bun.IDB, allocation *shipment.ChargeAllocation) error {
	if _, err := tx.NewInsert().Model(allocation).Returning("*").Exec(ctx); err != nil {
		return fmt.Errorf("insert charge allocation %s: %w", allocation.ID, err)
	}

	return nil
}

func (r *repository) update(
	ctx context.Context,
	tx bun.IDB,
	allocation, existing *shipment.ChargeAllocation,
) error {
	if existing.IsInvoiced() && (existing.BillToCustomerID != allocation.BillToCustomerID ||
		existing.Method != allocation.Method ||
		!existing.Percent.Decimal.Equal(allocation.Percent.Decimal) ||
		!existing.Amount.Decimal.Equal(allocation.Amount.Decimal)) {
		return errortypes.NewValidationError(
			"chargeAllocations",
			errortypes.ErrInvalidOperation,
			"An allocation already carried on an invoice cannot change",
		)
	}

	ov := existing.Version
	allocation.Version = ov + 1
	allocation.UpdatedAt = timeutils.NowUnix()
	allocation.InvoiceID = existing.InvoiceID
	allocation.InvoicedAt = existing.InvoicedAt

	cols := buncolgen.ChargeAllocationColumns
	result, err := tx.NewUpdate().
		Model(allocation).
		Column(
			cols.BillToCustomerID.Bare(),
			cols.Method.Bare(),
			cols.Percent.Bare(),
			cols.Amount.Bare(),
			cols.Sequence.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update charge allocation %s: %w", allocation.ID, err)
	}

	return dberror.CheckRowsAffected(result, "Charge allocation", allocation.ID.String())
}

func (r *repository) existingForShipment(
	ctx context.Context,
	tx bun.IDB,
	ti pagination.TenantInfo,
	shipmentID pulid.ID,
) (map[pulid.ID]*shipment.ChargeAllocation, error) {
	cols := buncolgen.ChargeAllocationColumns
	rows := make([]*shipment.ChargeAllocation, 0)
	if err := tx.NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ChargeAllocationScopeTenant(sq, ti).
				Where(cols.ShipmentID.Eq(), shipmentID)
		}).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get existing charge allocations: %w", err)
	}

	result := make(map[pulid.ID]*shipment.ChargeAllocation, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}

	return result, nil
}

// followCustomerChange moves uninvoiced rows that named the shipment's previous
// customer as payer onto the new one, so "the customer pays" keeps meaning the
// customer after a reassignment.
func (r *repository) followCustomerChange(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	if entity.PreviousCustomerID.IsNil() || entity.PreviousCustomerID == entity.CustomerID {
		return nil
	}

	cols := buncolgen.ChargeAllocationColumns
	ti := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	if _, err := tx.NewUpdate().
		Model((*shipment.ChargeAllocation)(nil)).
		Set(cols.BillToCustomerID.Set(), entity.CustomerID).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ChargeAllocationScopeTenantUpdate(uq, ti).
				Where(cols.ShipmentID.Eq(), entity.ID).
				Where(cols.BillToCustomerID.Eq(), entity.PreviousCustomerID).
				Where(cols.InvoiceID.IsNull())
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("follow customer change on charge allocations: %w", err)
	}

	return nil
}

// dropOrphans removes accessorial allocations whose charge is gone. The charge
// delete cascades in the database, but a charge replaced in the same save keeps
// the row alive only if the payload re-targets it, which normalize enforces.
func (r *repository) dropOrphans(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	cols := buncolgen.ChargeAllocationColumns
	ac := buncolgen.AdditionalChargeColumns
	ti := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	if _, err := tx.NewDelete().
		Model((*shipment.ChargeAllocation)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ChargeAllocationScopeTenantDelete(dq, ti).
				Where(cols.ShipmentID.Eq(), entity.ID).
				Where(cols.ChargeKind.Eq(), shipment.ChargeAllocationKindAccessorial).
				Where(cols.InvoiceID.IsNull()).
				Where(`NOT EXISTS (SELECT 1 FROM additional_charges AS ac WHERE ` +
					ac.ID.Qualified() + ` = ` + cols.AdditionalChargeID.Qualified() + `)`)
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("drop orphaned charge allocations: %w", err)
	}

	return nil
}

func (r *repository) SyncForOrderCharge(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.SyncOrderChargeAllocationsRequest,
) error {
	if req == nil || req.Allocations == nil {
		return nil
	}
	cols := buncolgen.ChargeAllocationColumns

	existingRows := make([]*shipment.ChargeAllocation, 0)
	if err := tx.NewSelect().
		Model(&existingRows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ChargeAllocationScopeTenant(sq, req.TenantInfo).
				Where(cols.OrderChargeID.Eq(), req.OrderChargeID)
		}).
		Scan(ctx); err != nil {
		return fmt.Errorf("get existing order charge allocations: %w", err)
	}
	existing := make(map[pulid.ID]*shipment.ChargeAllocation, len(existingRows))
	for _, row := range existingRows {
		existing[row.ID] = row
	}

	kept := make(map[pulid.ID]struct{}, len(req.Allocations))
	for _, allocation := range req.Allocations {
		if allocation == nil {
			continue
		}
		allocation.OrganizationID = req.TenantInfo.OrgID
		allocation.BusinessUnitID = req.TenantInfo.BuID
		allocation.ChargeKind = shipment.ChargeAllocationKindOrderCharge
		chargeID := req.OrderChargeID
		allocation.OrderChargeID = &chargeID
		allocation.ShipmentID = nil
		allocation.AdditionalChargeID = nil
		allocation.AdditionalChargeIndex = nil

		var err error
		switch {
		case allocation.ID.IsNil():
			allocation.ID = pulid.MustNew("chal_")
			err = r.insert(ctx, tx, allocation)
		case existing[allocation.ID] != nil:
			err = r.update(ctx, tx, allocation, existing[allocation.ID])
		default:
			err = errortypes.NewBusinessError("Order charge contains an unknown allocation").
				WithParam("chargeAllocationId", allocation.ID.String())
		}
		if err != nil {
			return err
		}
		kept[allocation.ID] = struct{}{}
	}

	deleteIDs := make([]pulid.ID, 0, len(existing))
	for id, row := range existing {
		if _, ok := kept[id]; ok {
			continue
		}
		if row.IsInvoiced() {
			return errortypes.NewValidationError(
				"allocations",
				errortypes.ErrInvalidOperation,
				"An allocation already carried on an invoice cannot be removed",
			)
		}
		deleteIDs = append(deleteIDs, id)
	}
	if len(deleteIDs) == 0 {
		return nil
	}

	if _, err := tx.NewDelete().
		Model((*shipment.ChargeAllocation)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ChargeAllocationScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(deleteIDs))
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete order charge allocations: %w", err)
	}

	return nil
}

func (r *repository) ListByShipmentIDs(
	ctx context.Context,
	req *repositories.ListChargeAllocationsByShipmentIDsRequest,
) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
	result := make(map[pulid.ID][]*shipment.ChargeAllocation, len(req.ShipmentIDs))
	if len(req.ShipmentIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.ChargeAllocationColumns
	rows := make([]*shipment.ChargeAllocation, 0, len(req.ShipmentIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Relation(buncolgen.ChargeAllocationRelations.BillToCustomer).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ChargeAllocationScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.In(), bun.List(req.ShipmentIDs))
		}).
		Order(cols.ShipmentID.OrderAsc(), cols.Sequence.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list charge allocations by shipment", zap.Error(err))
		return nil, fmt.Errorf("list charge allocations by shipment: %w", err)
	}

	for _, row := range rows {
		if row.ShipmentID == nil {
			continue
		}
		result[*row.ShipmentID] = append(result[*row.ShipmentID], row)
	}

	return result, nil
}

func (r *repository) ListByOrderChargeIDs(
	ctx context.Context,
	req *repositories.ListChargeAllocationsByOrderChargeIDsRequest,
) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
	result := make(map[pulid.ID][]*shipment.ChargeAllocation, len(req.OrderChargeIDs))
	if len(req.OrderChargeIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.ChargeAllocationColumns
	rows := make([]*shipment.ChargeAllocation, 0, len(req.OrderChargeIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Relation(buncolgen.ChargeAllocationRelations.BillToCustomer).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ChargeAllocationScopeTenant(sq, req.TenantInfo).
				Where(cols.OrderChargeID.In(), bun.List(req.OrderChargeIDs))
		}).
		Order(cols.OrderChargeID.OrderAsc(), cols.Sequence.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list charge allocations by order charge", zap.Error(err))
		return nil, fmt.Errorf("list charge allocations by order charge: %w", err)
	}

	for _, row := range rows {
		if row.OrderChargeID == nil {
			continue
		}
		result[*row.OrderChargeID] = append(result[*row.OrderChargeID], row)
	}

	return result, nil
}

func (r *repository) MarkInvoiced(
	ctx context.Context,
	req *repositories.MarkChargeAllocationsInvoicedRequest,
) (int64, error) {
	if req == nil || len(req.AllocationIDs) == 0 || req.InvoiceID.IsNil() {
		return 0, nil
	}

	cols := buncolgen.ChargeAllocationColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*shipment.ChargeAllocation)(nil)).
		Set(cols.InvoiceID.Set(), req.InvoiceID).
		Set(cols.InvoicedAt.Set(), req.InvoicedAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ChargeAllocationScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.AllocationIDs)).
				Where(cols.InvoiceID.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("mark charge allocations invoiced: %w", err)
	}

	return result.RowsAffected()
}

func (r *repository) ClearInvoice(
	ctx context.Context,
	req *repositories.ClearChargeAllocationsInvoiceRequest,
) (int64, error) {
	if req == nil || req.InvoiceID.IsNil() {
		return 0, nil
	}

	cols := buncolgen.ChargeAllocationColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*shipment.ChargeAllocation)(nil)).
		Set(cols.InvoiceID.SetNull()).
		Set(cols.InvoicedAt.SetNull()).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ChargeAllocationScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.InvoiceID.Eq(), req.InvoiceID)
		}).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("clear charge allocation invoice: %w", err)
	}

	return result.RowsAffected()
}

func (r *repository) LockedIDs(
	ctx context.Context,
	ti pagination.TenantInfo,
	allocationIDs []pulid.ID,
) (map[pulid.ID]struct{}, error) {
	locked := make(map[pulid.ID]struct{})
	if len(allocationIDs) == 0 {
		return locked, nil
	}

	cols := buncolgen.ChargeAllocationColumns
	inv := buncolgen.InvoiceColumns
	ids := make([]pulid.ID, 0, len(allocationIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.ChargeAllocation)(nil)).
		Column(cols.ID.Bare()).
		Join("JOIN invoices AS inv ON "+inv.ID.Qualified()+" = "+cols.InvoiceID.Qualified()).
		JoinOn(inv.OrganizationID.Qualified()+" = "+cols.OrganizationID.Qualified()).
		JoinOn(inv.BusinessUnitID.Qualified()+" = "+cols.BusinessUnitID.Qualified()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ChargeAllocationScopeTenant(sq, ti).
				Where(cols.ID.In(), bun.List(allocationIDs)).
				Where(inv.Status.Eq(), invoice.StatusPosted)
		}).
		Scan(ctx, &ids); err != nil {
		return nil, fmt.Errorf("find locked charge allocations: %w", err)
	}

	for _, id := range ids {
		locked[id] = struct{}{}
	}

	return locked, nil
}

// RefreshQueueSnapshots rewrites each open queue item's allocated total from the
// shipment's current charges. The queue list, statement previews and the
// reconciliation check all read the snapshot rather than recomputing shares.
func (r *repository) RefreshQueueSnapshots(
	ctx context.Context,
	tx bun.IDB,
	ti pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	if shipmentID.IsNil() {
		return nil
	}

	sp := buncolgen.ShipmentColumns
	shp := new(shipment.Shipment)
	if err := tx.NewSelect().
		Model(shp).
		Relation(buncolgen.ShipmentRelations.AdditionalCharges).
		Relation(buncolgen.ShipmentRelations.ChargeAllocations).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, ti).Where(sp.ID.Eq(), shipmentID)
		}).
		Scan(ctx); err != nil {
		return fmt.Errorf("load shipment for allocation snapshot: %w", err)
	}

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	if err != nil {
		return err
	}

	bqi := buncolgen.BillingQueueItemColumns
	for _, share := range resolution.Shares {
		if _, err = tx.NewUpdate().
			Model((*billingqueue.BillingQueueItem)(nil)).
			Set(bqi.AllocatedTotalAmount.Set(), share.TotalAmount).
			Set(bqi.AllocatedTotalAmountMinor.Set(), money.MinorUnits(share.TotalAmount)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.BillingQueueItemScopeTenantUpdate(uq, ti).
					Where(bqi.ShipmentID.Eq(), shipmentID).
					Where(bqi.BillToCustomerID.Eq(), share.PayerID).
					Where(bqi.BillType.Eq(), billingqueue.BillTypeInvoice).
					Where(bqi.Status.NotIn(), bun.List([]billingqueue.Status{
						billingqueue.StatusPosted,
						billingqueue.StatusCanceled,
					}))
			}).
			Exec(ctx); err != nil {
			return fmt.Errorf("refresh billing queue allocated totals: %w", err)
		}
	}

	return nil
}
