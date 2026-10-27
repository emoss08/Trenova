package dispatchcontrolrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dispatchControlErrors = dberror.Mapping{Entity: "DispatchControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dispatchcontrolports.DispatchControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-control-repository"),
	}
}

func (r *repository) find(ctx context.Context) (*dispatchcontrol.DispatchControl, error) {
	return dispatchcontroldb.DispatchControls.Cache(resultcache.Settings()...).
		One(ctx, r.db.Keel(ctx))
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*dispatchcontrol.DispatchControl, error) {
			entity, err := r.find(ctx)
			if err != nil {
				r.l.Error(
					"failed to get dispatch control",
					zap.String("orgId", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return nil, dispatchControlErrors.Map(err, req.TenantInfo.OrgID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*dispatchcontrol.DispatchControl, error) {
			err := dispatchcontroldb.DispatchControls.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create dispatch control",
					zap.String("orgId", entity.OrganizationID.String()),
					zap.Error(err),
				)
				return nil, dispatchControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*dispatchcontrol.DispatchControl, error) {
			err := dispatchcontroldb.DispatchControls.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update dispatch control",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, dispatchControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dispatchcontrol.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "GetOrCreate"),
		zap.String("orgId", orgID.String()),
	)

	dc := dispatchcontroldb.DispatchControls
	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*dispatchcontrol.DispatchControl]{
		Find: r.find,
		Create: func(ctx context.Context) error {
			_, err := dc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*dispatchcontrol.DispatchControl{
					dispatchcontrol.NewDefaultDispatchControl(orgID, buID),
				},
				dc.OrganizationID,
				dc.BusinessUnitID,
			)
			if err != nil {
				log.Error("failed to create default dispatch control", zap.Error(err))
				return dberror.MapRetryableTransactionError(
					err,
					"Dispatch control is busy. Retry the request.",
				)
			}

			return nil
		},
		Default: func() *dispatchcontrol.DispatchControl {
			return dispatchcontrol.NewDefaultDispatchControl(orgID, buID)
		},
		MapError: func(err error) error {
			log.Error("failed to get dispatch control", zap.Error(err))
			return dispatchControlErrors.Map(err, orgID.String())
		},
	})
}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListHorizonPlanningTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with horizon planning on")
	dc := dispatchcontroldb.DispatchControls
	return r.listTenantsWhere(
		ctx,
		dc.EnableAutoAssignment.IsTrue(),
		dc.PlanningMode.Eq(dispatchcontrol.PlanningModeHorizon),
	)
}

func (r *repository) ListCheckCallMonitoringTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that watch brokered loads for check calls")
	return r.listTenantsWhere(
		ctx,
		dispatchcontroldb.DispatchControls.EnableCheckCallMonitoring.IsTrue(),
	)
}

func (r *repository) listTenantsWhere(
	ctx context.Context,
	preds ...keel.Pred,
) ([]pagination.TenantInfo, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		dc := dispatchcontroldb.DispatchControls
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), dc.AllTenants().
			Pick(dc.OrganizationID, dc.BusinessUnitID).
			Where(preds...).
			Order(dc.OrganizationID.Asc()))
		if err != nil {
			r.l.Error("failed to list dispatch control tenants", zap.Error(err))
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
