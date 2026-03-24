package shipmentholdrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.ShipmentHoldRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-hold-repository"),
	}
}

func (r *repository) ListByShipmentID(
	ctx context.Context,
	req *repositories.ListShipmentHoldsRequest,
) (*pagination.ListResult[*shipment.ShipmentHold], error) {
	if req.Filter.Pagination.Limit <= 0 {
		req.Filter.Pagination.Limit = 50
	}

	shh := buncolgen.ShipmentHoldColumns
	db := r.db.DBForContext(ctx)
	holds := make([]*shipment.ShipmentHold, 0, req.Filter.Pagination.Limit)

	total, err := db.NewSelect().
		Model(&holds).
		Where(shh.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentHoldApplyTenant(req.Filter.TenantInfo)).
		Relation(buncolgen.ShipmentHoldRelations.HoldReason).
		Relation(buncolgen.ShipmentHoldRelations.CreatedBy).
		Relation(buncolgen.ShipmentHoldRelations.ReleasedBy).
		OrderExpr(shh.ReleasedAt.Expr("{} IS NULL DESC")).
		Order(shh.StartedAt.OrderDesc()).
		OrderExpr(shh.ReleasedAt.Expr("COALESCE({}, 0) DESC")).
		Limit(req.Filter.Pagination.Limit).
		Offset(req.Filter.Pagination.Offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*shipment.ShipmentHold]{
		Items: holds,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	shh := buncolgen.ShipmentHoldColumns
	db := r.db.DBForContext(ctx)
	entity := new(shipment.ShipmentHold)

	if err := db.NewSelect().
		Model(entity).
		Where(shh.ID.Eq(), req.HoldID).
		Where(shh.ShipmentID.Eq(), req.ShipmentID).
		Apply(buncolgen.ShipmentHoldApplyTenant(req.TenantInfo)).
		Relation(buncolgen.ShipmentHoldRelations.HoldReason).
		Relation(buncolgen.ShipmentHoldRelations.CreatedBy).
		Relation(buncolgen.ShipmentHoldRelations.ReleasedBy).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment hold")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.ShipmentHold,
) (*shipment.ShipmentHold, error) {
	db := r.db.DBForContext(ctx)
	if _, err := db.NewInsert().Model(entity).Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) &&
			dberror.ExtractConstraintName(err) == "ux_shipment_holds_active_by_type" {
			return nil, errortypes.NewBusinessError("Shipment already has an active hold of this type").
				WithParam("shipmentId", entity.ShipmentID.String())
		}
		return nil, fmt.Errorf("insert shipment hold: %w", err)
	}

	return r.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.ShipmentHold,
) (*shipment.ShipmentHold, error) {
	shh := buncolgen.ShipmentHoldColumns
	db := r.db.DBForContext(ctx)
	result, err := db.NewUpdate().
		Model(entity).
		Where(shh.ID.Eq(), entity.ID).
		Where(shh.ShipmentID.Eq(), entity.ShipmentID).
		Where(shh.OrganizationID.Eq(), entity.OrganizationID).
		Where(shh.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(shh.ReleasedAt.IsNull()).
		Where(shh.Version.Eq(), entity.Version).
		Set(shh.StartedAt.Set(), entity.StartedAt).
		Set(shh.Severity.Set(), entity.Severity).
		Set(shh.Notes.Set(), entity.Notes).
		Set(shh.BlocksDispatch.Set(), entity.BlocksDispatch).
		Set(shh.BlocksDelivery.Set(), entity.BlocksDelivery).
		Set(shh.BlocksBilling.Set(), entity.BlocksBilling).
		Set(shh.VisibleToCustomer.Set(), entity.VisibleToCustomer).
		Set(shh.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update shipment hold: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, "Shipment hold", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Release(
	ctx context.Context,
	entity *shipment.ShipmentHold,
) (*shipment.ShipmentHold, error) {
	shh := buncolgen.ShipmentHoldColumns
	db := r.db.DBForContext(ctx)
	result, err := db.NewUpdate().
		Model((*shipment.ShipmentHold)(nil)).
		Where(shh.ID.Eq(), entity.ID).
		Where(shh.ShipmentID.Eq(), entity.ShipmentID).
		Where(shh.OrganizationID.Eq(), entity.OrganizationID).
		Where(shh.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(shh.ReleasedAt.IsNull()).
		Set(shh.ReleasedAt.Set(), entity.ReleasedAt).
		Set(shh.ReleasedByID.Set(), entity.ReleasedByID).
		Set(shh.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("release shipment hold: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, "Shipment hold", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) HasActiveDispatchHold(
	ctx context.Context,
	req *repositories.ActiveShipmentHoldRequest,
) (bool, error) {
	return r.hasActiveBlockingHold(ctx, req, "blocks_dispatch")
}

func (r *repository) HasActiveDeliveryHold(
	ctx context.Context,
	req *repositories.ActiveShipmentHoldRequest,
) (bool, error) {
	return r.hasActiveBlockingHold(ctx, req, "blocks_delivery")
}

func (r *repository) hasActiveBlockingHold(
	ctx context.Context,
	req *repositories.ActiveShipmentHoldRequest,
	column string,
) (bool, error) {
	shh := buncolgen.ShipmentHoldColumns
	db := r.db.DBForContext(ctx)
	count, err := db.NewSelect().
		Model((*shipment.ShipmentHold)(nil)).
		Where(shh.ShipmentID.Eq(), req.ShipmentID).
		Where(shh.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(shh.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(shh.ReleasedAt.IsNull()).
		Where(shh.Severity.Eq(), "Blocking").
		Where("? = TRUE", bun.Ident(column)).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("check active shipment hold: %w", err)
	}

	return count > 0, nil
}

func tenantInfo(entity *shipment.ShipmentHold) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
