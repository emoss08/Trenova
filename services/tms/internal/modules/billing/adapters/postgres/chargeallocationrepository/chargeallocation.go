package chargeallocationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var allocationErrors = dberror.Mapping{Entity: "Charge allocation"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) chargeallocationports.ChargeAllocationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.charge-allocation-repository"),
	}
}

func (r *repository) SyncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	if entity == nil {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if entity.ChargeAllocations != nil {
			if err := r.syncShipmentRows(ctx, entity); err != nil {
				return err
			}
		}

		if err := r.followCustomerChange(ctx, entity); err != nil {
			return err
		}

		if err := r.dropOrphans(ctx, entity); err != nil {
			return err
		}

		return r.RefreshQueueSnapshots(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		}, entity.ID)
	})
}

func (r *repository) syncShipmentRows(ctx context.Context, entity *shipment.Shipment) error {
	ca := shipmentdb.ChargeAllocations
	existing, err := r.existing(ctx, ca.ShipmentID.Eq(entity.ID))
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
			if err = r.insert(ctx, allocation); err != nil {
				return err
			}
		case existing[allocation.ID] != nil:
			if err = r.update(ctx, allocation, existing[allocation.ID]); err != nil {
				return err
			}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown charge allocation").
				WithParam("chargeAllocationId", allocation.ID.String())
		}
		kept[allocation.ID] = struct{}{}
	}

	return r.deleteUnkept(ctx, existing, kept, "chargeAllocations")
}

func (r *repository) deleteUnkept(
	ctx context.Context,
	existing map[pulid.ID]*shipment.ChargeAllocation,
	kept map[pulid.ID]struct{},
	field string,
) error {
	deleteIDs := make([]pulid.ID, 0, len(existing))
	for id, row := range existing {
		if _, ok := kept[id]; ok {
			continue
		}
		if row.IsInvoiced() {
			return errortypes.NewValidationError(
				field,
				errortypes.ErrInvalidOperation,
				"An allocation already carried on an invoice cannot be removed",
			)
		}
		deleteIDs = append(deleteIDs, id)
	}
	if len(deleteIDs) == 0 {
		return nil
	}

	ca := shipmentdb.ChargeAllocations
	_, err := ca.Where(ca.ID.In(deleteIDs...)).Delete(ctx, r.db.Keel(ctx))
	return allocationErrors.Map(err, "")
}

func (r *repository) normalize(
	entity *shipment.Shipment,
	allocation *shipment.ChargeAllocation,
) error {
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
			if idx < 0 || idx >= len(entity.AdditionalCharges) ||
				entity.AdditionalCharges[idx] == nil ||
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

func (r *repository) insert(ctx context.Context, allocation *shipment.ChargeAllocation) error {
	err := shipmentdb.ChargeAllocations.Create(ctx, r.db.Keel(ctx), allocation)
	return allocationErrors.Map(err, allocation.ID.String())
}

func (r *repository) update(
	ctx context.Context,
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

	allocation.Version = existing.Version
	allocation.InvoiceID = existing.InvoiceID
	allocation.InvoicedAt = existing.InvoicedAt

	ca := shipmentdb.ChargeAllocations
	err := ca.Save(ctx, r.db.Keel(ctx), allocation,
		ca.BillToCustomerID,
		ca.Method,
		ca.Percent,
		ca.Amount,
		ca.Sequence,
	)

	return allocationErrors.Map(err, allocation.ID.String())
}

func (r *repository) existing(
	ctx context.Context,
	scope keel.Pred,
) (map[pulid.ID]*shipment.ChargeAllocation, error) {
	rows, err := shipmentdb.ChargeAllocations.Where(scope).All(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, err
	}

	result := make(map[pulid.ID]*shipment.ChargeAllocation, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}

	return result, nil
}

func (r *repository) followCustomerChange(ctx context.Context, entity *shipment.Shipment) error {
	if entity.PreviousCustomerID.IsNil() || entity.PreviousCustomerID == entity.CustomerID {
		return nil
	}

	ca := shipmentdb.ChargeAllocations
	_, err := ca.Where(
		ca.ShipmentID.Eq(entity.ID),
		ca.BillToCustomerID.Eq(entity.PreviousCustomerID),
		ca.InvoiceID.IsNull(),
	).Update(ctx, r.db.Keel(ctx),
		ca.BillToCustomerID.To(entity.CustomerID),
		ca.Version.ToExpr(ca.Version),
	)

	return allocationErrors.Map(err, "")
}

func (r *repository) dropOrphans(ctx context.Context, entity *shipment.Shipment) error {
	ca := shipmentdb.ChargeAllocations
	ac := shipmentdb.AdditionalCharges
	_, err := ca.Where(
		ca.ShipmentID.Eq(entity.ID),
		ca.ChargeKind.Eq(shipment.ChargeAllocationKindAccessorial),
		ca.InvoiceID.IsNull(),
		keel.NotExists(ac.Where(ac.ID.EqCol(ca.AdditionalChargeID))),
	).Delete(ctx, r.db.Keel(ctx))

	return allocationErrors.Map(err, "")
}

func (r *repository) SyncForOrderCharge(
	ctx context.Context,
	req *chargeallocationports.SyncOrderChargeAllocationsRequest,
) error {
	if req == nil || req.Allocations == nil {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ca := shipmentdb.ChargeAllocations
		existing, err := r.existing(ctx, ca.OrderChargeID.Eq(req.OrderChargeID))
		if err != nil {
			return err
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

			switch {
			case allocation.ID.IsNil():
				err = r.insert(ctx, allocation)
			case existing[allocation.ID] != nil:
				err = r.update(ctx, allocation, existing[allocation.ID])
			default:
				err = errortypes.NewBusinessError("Order charge contains an unknown allocation").
					WithParam("chargeAllocationId", allocation.ID.String())
			}
			if err != nil {
				return err
			}
			kept[allocation.ID] = struct{}{}
		}

		return r.deleteUnkept(ctx, existing, kept, "allocations")
	})
}

func (r *repository) ListByShipmentIDs(
	ctx context.Context,
	req *chargeallocationports.ListChargeAllocationsByShipmentIDsRequest,
) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			result := make(map[pulid.ID][]*shipment.ChargeAllocation, len(req.ShipmentIDs))
			if len(req.ShipmentIDs) == 0 {
				return result, nil
			}

			ca := shipmentdb.ChargeAllocations
			rows, err := ca.Where(ca.ShipmentID.In(req.ShipmentIDs...)).
				Load(ca.BillToCustomer()).
				Order(ca.ShipmentID.Asc(), ca.Sequence.Asc(), ca.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list charge allocations by shipment", zap.Error(err))
				return nil, err
			}

			for _, row := range rows {
				if row.ShipmentID == nil {
					continue
				}
				result[*row.ShipmentID] = append(result[*row.ShipmentID], row)
			}

			return result, nil
		},
	)
}

func (r *repository) ListByOrderChargeIDs(
	ctx context.Context,
	req *chargeallocationports.ListChargeAllocationsByOrderChargeIDsRequest,
) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*shipment.ChargeAllocation, error) {
			result := make(map[pulid.ID][]*shipment.ChargeAllocation, len(req.OrderChargeIDs))
			if len(req.OrderChargeIDs) == 0 {
				return result, nil
			}

			ca := shipmentdb.ChargeAllocations
			rows, err := ca.Where(ca.OrderChargeID.In(req.OrderChargeIDs...)).
				Load(ca.BillToCustomer()).
				Order(ca.OrderChargeID.Asc(), ca.Sequence.Asc(), ca.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list charge allocations by order charge", zap.Error(err))
				return nil, err
			}

			for _, row := range rows {
				if row.OrderChargeID == nil {
					continue
				}
				result[*row.OrderChargeID] = append(result[*row.OrderChargeID], row)
			}

			return result, nil
		},
	)
}

func (r *repository) MarkInvoiced(
	ctx context.Context,
	req *chargeallocationports.MarkChargeAllocationsInvoicedRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil || len(req.AllocationIDs) == 0 || req.InvoiceID.IsNil() {
			return 0, nil
		}

		ca := shipmentdb.ChargeAllocations
		n, err := ca.Where(ca.ID.In(req.AllocationIDs...), ca.InvoiceID.IsNull()).
			Update(ctx, r.db.Keel(ctx),
				ca.InvoiceID.To(req.InvoiceID),
				ca.InvoicedAt.To(req.InvoicedAt),
				ca.Version.ToExpr(ca.Version),
			)
		if err != nil {
			return 0, allocationErrors.Map(err, req.InvoiceID.String())
		}

		return n, nil
	})
}

func (r *repository) ClearInvoice(
	ctx context.Context,
	req *chargeallocationports.ClearChargeAllocationsInvoiceRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil || req.InvoiceID.IsNil() {
			return 0, nil
		}

		ca := shipmentdb.ChargeAllocations
		n, err := ca.Where(ca.InvoiceID.Eq(req.InvoiceID)).Update(ctx, r.db.Keel(ctx),
			ca.InvoiceID.ToNull(),
			ca.InvoicedAt.ToNull(),
			ca.Version.ToExpr(ca.Version),
		)
		if err != nil {
			return 0, allocationErrors.Map(err, req.InvoiceID.String())
		}

		return n, nil
	})
}

func (r *repository) LockedIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	allocationIDs []pulid.ID,
) (map[pulid.ID]struct{}, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]struct{}, error) {
		locked := make(map[pulid.ID]struct{})
		if len(allocationIDs) == 0 {
			return locked, nil
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), lockedAllocations(allocationIDs))
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			locked[id] = struct{}{}
		}

		return locked, nil
	})
}

func lockedAllocations(allocationIDs []pulid.ID) *keel.Query[*shipment.ChargeAllocation] {
	ca := shipmentdb.ChargeAllocations
	inv := invoicedb.Invoices
	return ca.Pick(ca.ID).
		Join(inv, inv.ID.EqCol(ca.InvoiceID)).
		Where(ca.ID.In(allocationIDs...), inv.Status.Eq(invoice.StatusPosted))
}

func (r *repository) RefreshQueueSnapshots(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	if shipmentID.IsNil() {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		sh := shipmentdb.Shipments
		shp, err := sh.Where(sh.ID.Eq(shipmentID)).
			Load(sh.AdditionalCharges(), sh.ChargeAllocations()).
			One(ctx, db)
		if err != nil {
			return dberror.HandleNotFoundError(err, "Shipment")
		}

		resolution, err := chargeallocation.ResolveShares(ctx, shp, shp.ChargeAllocations)
		if err != nil {
			return err
		}

		bq := billingqueuedb.BillingQueueItems
		for _, share := range resolution.Shares {
			if _, err = bq.Where(
				bq.ShipmentID.Eq(shipmentID),
				bq.BillToCustomerID.Eq(share.PayerID),
				bq.BillType.Eq(billingqueue.BillTypeInvoice),
				bq.Status.NotIn(billingqueue.StatusPosted, billingqueue.StatusCanceled),
			).Update(ctx, db,
				bq.AllocatedTotalAmount.To(share.TotalAmount),
				bq.AllocatedTotalAmountMinor.To(money.MinorUnits(share.TotalAmount)),
				bq.Version.ToExpr(bq.Version),
				bq.UpdatedAt.ToExpr(bq.UpdatedAt),
			); err != nil {
				return err
			}
		}

		return nil
	})
}
