package shipmentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.ShipmentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipmentcontrol-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetShipmentControlRequest,
) (*tenant.ShipmentControl, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
		zap.String("buID", req.TenantInfo.BuID.String()),
	)

	entity := new(tenant.ShipmentControl)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Where("sc.organization_id = ?", req.TenantInfo.OrgID).
		Where("sc.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx); err != nil {
		log.Error("failed to get shipment control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ShipmentControl")
	}

	return entity, nil
}

// ListDetentionEngineTenants enumerates the tenants that have opted into the
// detention policy engine, so tenant-fanning sweeps only visit organizations
// where the engine can produce work.
func (r *repository) ListDetentionEngineTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	rows := make([]tenant.ShipmentControl, 0, 16)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Column("sc.organization_id", "sc.business_unit_id").
		Where("sc.use_detention_policy_engine = TRUE").
		Order("sc.organization_id ASC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list detention engine tenants", zap.Error(err))
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, len(rows))
	for i := range rows {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: rows[i].OrganizationID,
			BuID:  rows[i].BusinessUnitID,
		})
	}

	return tenants, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.ShipmentControl,
) (*tenant.ShipmentControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update shipment control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "ShipmentControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
