package shipmentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var shipmentControlErrors = dberror.Mapping{Entity: "ShipmentControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.ShipmentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipmentcontrol-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req tenantports.GetShipmentControlRequest,
) (*tenant.ShipmentControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.ShipmentControl, error) {
		entity, err := tenantdb.ShipmentControls.Cache(resultcache.Settings()...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentControlErrors.Map(err, req.TenantInfo.OrgID.String())
		}

		return entity, nil
	})
}

// ListDetentionEngineTenants enumerates the tenants that have opted into the
// detention policy engine, so tenant-fanning sweeps only visit organizations
// where the engine can produce work.
func (r *repository) ListDetentionEngineTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with the detention engine on")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		sc := tenantdb.ShipmentControls
		q := sc.AllTenants().
			Pick(sc.OrganizationID, sc.BusinessUnitID).
			Where(sc.UseDetentionPolicyEngine.IsTrue()).
			Order(sc.OrganizationID.Asc())
		if limit > 0 {
			q = q.Limit(limit)
		}

		rows, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list detention engine tenants", zap.Error(err))
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.ShipmentControl,
) (*tenant.ShipmentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.ShipmentControl, error) {
		if err := tenantdb.ShipmentControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update shipment control",
				zap.String("orgID", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, shipmentControlErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
