package tractorrepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/lastknownlocation"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var tractorErrors = dberror.Mapping{
	Entity: "Tractor",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_tractors_code",
			Field:   "code",
			Message: "Tractor with this code already exists in your organization",
		},
		{
			Name:    "idx_tractors_primary_worker_id",
			Field:   "primaryWorkerId",
			Message: "Primary worker is already assigned to another tractor",
		},
		{
			Name:    "idx_tractors_external_id_tenant",
			Field:   "externalId",
			Message: "Tractor with this external ID already exists in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) tractorports.TractorRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.tractor-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func (r *repository) statusFilter(value string) keel.Pred {
	if value == "" {
		return nil
	}

	status, err := domaintypes.EquipmentStatusFromString(value)
	if err != nil {
		r.l.Error("failed to parse equipment status", zap.Error(err))
		return nil
	}

	return tractordb.Tractors.Status.Eq(status)
}

func (r *repository) List(
	ctx context.Context,
	req *tractorports.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tractor.Tractor], error) {
			loads, err := tractorLoads(&req.TractorRelationIncludes)
			if err != nil {
				return nil, err
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				tractordb.Tractors.Where(r.statusFilter(req.Status)),
				&listquery.Request[tractor.Tractor]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.TractorColumns,
					Computed: map[string]keel.Expr{
						lastKnownLocationFilter: lastknownlocation.LocationID(
							tractordb.Tractors.ID,
							equipmentcontinuity.EquipmentTypeTractor,
						),
					},
					Shape: func(q *keel.Query[tractor.Tractor]) *keel.Query[tractor.Tractor] {
						return withLastKnownLocation(q.Load(loads...), &req.TractorRelationIncludes)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list tractors", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			return tractordb.Tractors.Create(c, r.db.Keel(c), entity)
		}, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterTractorsTotal,
			Quantity: 1,
		})
		if err != nil {
			r.l.Error("failed to create tractor", zap.String("code", entity.Code), zap.Error(err))
			return nil, tractorErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		if err := tractordb.Tractors.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update tractor",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, tractorErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req tractorports.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		t := tractordb.Tractors
		q, err := tractorQuery(&req.TractorRelationIncludes)
		if err != nil {
			return nil, err
		}

		entity, err := q.Where(t.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, tractorErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req tractorports.GetTractorsByIDsRequest,
) ([]*tractor.Tractor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tractor.Tractor, error) {
		if len(req.TractorIDs) == 0 {
			return []*tractor.Tractor{}, nil
		}

		t := tractordb.Tractors
		q, err := tractorQuery(&req.TractorRelationIncludes)
		if err != nil {
			return nil, err
		}

		entities, err := q.Where(t.ID.In(req.TractorIDs...)).
			Limit(len(req.TractorIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get tractors", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *tractorports.BulkUpdateTractorStatusRequest,
) ([]*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*tractor.Tractor, error) {
		t := tractordb.Tractors
		entities, err := t.Where(t.ID.In(req.TractorIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), t.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update tractor status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "Tractor", req.TractorIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *tractorports.TractorSelectOptionsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tractor.Tractor], error) {
			t := tractordb.Tractors
			q := t.Pick(t.ID, t.CreatedAt, t.Status, t.Code, t.PrimaryWorkerID, t.SecondaryWorkerID).
				Where(t.Status.Eq(domaintypes.EquipmentStatusAvailable)).
				Load(t.PrimaryWorker(), t.SecondaryWorker())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectOptionsRequest,
				func(s string) keel.Pred { return t.Code.ContainsFold(s) },
			)
		},
	)
}

func tractorQuery(
	includes *tractorports.TractorRelationIncludes,
) (*keel.Query[tractor.Tractor], error) {
	loads, err := tractorLoads(includes)
	if err != nil {
		return nil, err
	}

	q, err := listquery.Pick(tractordb.Tractors.Query(), includes.TractorColumns)
	if err != nil {
		return nil, err
	}

	return withLastKnownLocation(q.Load(loads...), includes), nil
}

func tractorLoads(includes *tractorports.TractorRelationIncludes) ([]keel.Loader, error) {
	t := tractordb.Tractors
	details := includes.IncludeTenantDetails
	org := t.Organization()
	et := t.EquipmentType()
	em := t.EquipmentManufacturer()
	fc := t.FleetCode()

	var loads listquery.Loads
	loads.Add(details || includes.IncludeBusinessUnit, func() (keel.Loader, error) {
		return t.BusinessUnit(), nil
	})
	loads.Add(details || includes.IncludeOrganization, func() (keel.Loader, error) {
		return listquery.Load(
			org.Query(),
			nil,
			listquery.When(details, org.BusinessUnit(), org.State())...)
	})
	loads.Add(includes.IncludeState, func() (keel.Loader, error) { return t.State(), nil })
	loads.Add(
		includes.IncludeEquipmentDetails || includes.IncludeEquipmentType,
		func() (keel.Loader, error) {
			return listquery.Load(et.Query(), includes.EquipmentTypeColumns,
				listquery.When(details, et.BusinessUnit(), et.Organization())...)
		},
	)
	loads.Add(
		includes.IncludeEquipmentDetails || includes.IncludeEquipmentManufacturer,
		func() (keel.Loader, error) {
			return listquery.Load(em.Query(), includes.EquipmentManufacturerColumns,
				listquery.When(details, em.BusinessUnit(), em.Organization())...)
		},
	)
	loads.Add(
		includes.IncludeFleetDetails || includes.IncludeFleetCode,
		func() (keel.Loader, error) {
			return listquery.Load(fc.Query(), includes.FleetCodeColumns,
				listquery.When(details, fc.BusinessUnit(), fc.Organization())...)
		},
	)
	pw := t.PrimaryWorker()
	loads.Add(
		includes.IncludeWorkerDetails || includes.IncludePrimaryWorker,
		func() (keel.Loader, error) {
			nested := slices.Concat(
				listquery.When(details, pw.BusinessUnit(), pw.Organization()),
				listquery.When(details || includes.IncludePrimaryWorkerState, pw.State()),
				listquery.When(includes.IncludePrimaryWorkerFleet, pw.FleetCode()),
				listquery.When(includes.IncludePrimaryWorkerManager, pw.Manager()),
			)
			return listquery.Load(pw.Query(), includes.PrimaryWorkerColumns, nested...)
		},
	)
	sw := t.SecondaryWorker()
	loads.Add(
		includes.IncludeWorkerDetails || includes.IncludeSecondaryWorker,
		func() (keel.Loader, error) {
			nested := slices.Concat(
				listquery.When(details, sw.BusinessUnit(), sw.Organization()),
				listquery.When(details || includes.IncludeSecondaryWorkerState, sw.State()),
				listquery.When(includes.IncludeSecondaryWorkerFleet, sw.FleetCode()),
				listquery.When(includes.IncludeSecondaryWorkerManager, sw.Manager()),
			)
			return listquery.Load(sw.Query(), includes.SecondaryWorkerColumns, nested...)
		},
	)
	loads.Add(includes.IncludeOwnerWorker, func() (keel.Loader, error) {
		return listquery.Load(t.OwnerWorker().Query(), includes.OwnerWorkerColumns)
	})

	return loads.Result()
}

const lastKnownLocationFilter = "lastKnownLocationId"

func withLastKnownLocation(
	q *keel.Query[tractor.Tractor],
	includes *tractorports.TractorRelationIncludes,
) *keel.Query[tractor.Tractor] {
	if !includes.IncludeLastKnownLocation {
		return q
	}

	t := tractordb.Tractors
	ec := lastknownlocation.Continuity()
	location := lastknownlocation.Of(ec, lastknownlocation.Tenant{
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
	})
	current := ec.Pick(ec.CurrentLocationID).
		Where(lastknownlocation.Current(ec, t.ID, equipmentcontinuity.EquipmentTypeTractor)...).
		Limit(1)

	return q.LeftJoinLateral(current).
		Compute(t.LastKnownLocationID, location.LocationID).
		Compute(t.LastKnownLocationName, location.Name).
		Compute(t.LastKnownLocationLatitude, location.Latitude).
		Compute(t.LastKnownLocationLongitude, location.Longitude)
}
