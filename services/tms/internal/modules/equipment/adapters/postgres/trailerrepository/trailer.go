package trailerrepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/lastknownlocation"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
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

var trailerErrors = dberror.Mapping{Entity: "Trailer"}

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

func New(p Params) trailerports.TrailerRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.trailer-repository"),
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

	return trailerdb.Trailers.Status.Eq(status)
}

func (r *repository) List(
	ctx context.Context,
	req *trailerports.ListTrailersRequest,
) (*pagination.CursorListResult[*trailer.Trailer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*trailer.Trailer], error) {
			loads, err := trailerLoads(&req.TrailerRelationIncludes)
			if err != nil {
				return nil, err
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				trailerdb.Trailers.Where(r.statusFilter(req.Status)),
				&listquery.Request[trailer.Trailer]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.TrailerColumns,
					Computed: map[string]keel.Expr{
						lastKnownLocationFilter: lastknownlocation.LocationID(
							trailerdb.Trailers.ID,
							equipmentcontinuity.EquipmentTypeTrailer,
						),
					},
					Shape: func(q *keel.Query[*trailer.Trailer]) *keel.Query[*trailer.Trailer] {
						return withLastKnownLocation(q.Load(loads...), &req.TrailerRelationIncludes)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list trailers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*trailer.Trailer, error) {
		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			return trailerdb.Trailers.Create(c, r.db.Keel(c), entity)
		}, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterTrailersTotal,
			Quantity: 1,
		})
		if err != nil {
			r.l.Error("failed to create trailer", zap.String("code", entity.Code), zap.Error(err))
			return nil, trailerErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*trailer.Trailer, error) {
		if err := trailerdb.Trailers.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update trailer",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, trailerErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, trailerports.GetTrailerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req trailerports.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*trailer.Trailer, error) {
		tr := trailerdb.Trailers
		q, err := trailerQuery(&req.TrailerRelationIncludes)
		if err != nil {
			return nil, err
		}

		entity, err := q.Where(tr.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, trailerErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*trailer.Trailer], error) {
			tr := trailerdb.Trailers
			q := tr.Pick(tr.ID, tr.CreatedAt, tr.Status, tr.Code).
				Where(tr.Status.Eq(domaintypes.EquipmentStatusAvailable))

			return listquery.SelectOptions(ctx, r.db.Keel(ctx), q, req, func(s string) keel.Pred {
				return tr.Code.ContainsFold(s)
			})
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req trailerports.GetTrailersByIDsRequest,
) ([]*trailer.Trailer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*trailer.Trailer, error) {
		if len(req.TrailerIDs) == 0 {
			return []*trailer.Trailer{}, nil
		}

		tr := trailerdb.Trailers
		q, err := trailerQuery(&req.TrailerRelationIncludes)
		if err != nil {
			return nil, err
		}

		entities, err := q.Where(tr.ID.In(req.TrailerIDs...)).
			Limit(len(req.TrailerIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get trailers", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *trailerports.BulkUpdateTrailerStatusRequest,
) ([]*trailer.Trailer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*trailer.Trailer, error) {
		tr := trailerdb.Trailers
		entities, err := tr.Where(tr.ID.In(req.TrailerIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), tr.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update trailer status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "Trailer", req.TrailerIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func trailerQuery(
	includes *trailerports.TrailerRelationIncludes,
) (*keel.Query[*trailer.Trailer], error) {
	loads, err := trailerLoads(includes)
	if err != nil {
		return nil, err
	}

	q, err := listquery.Pick(trailerdb.Trailers.Query(), includes.TrailerColumns)
	if err != nil {
		return nil, err
	}

	return withLastKnownLocation(q.Load(loads...), includes), nil
}

func trailerLoads(includes *trailerports.TrailerRelationIncludes) ([]keel.Loader, error) {
	tr := trailerdb.Trailers
	details := includes.IncludeTenantDetails
	org := tr.Organization()
	et := tr.EquipmentType()
	em := tr.EquipmentManufacturer()
	fc := tr.FleetCode()

	var loads listquery.Loads
	loads.Add(details || includes.IncludeBusinessUnit, func() (keel.Loader, error) {
		return tr.BusinessUnit(), nil
	})
	loads.Add(details || includes.IncludeOrganization, func() (keel.Loader, error) {
		return listquery.Load(
			org.Query(),
			nil,
			listquery.When(details, org.BusinessUnit(), org.State())...)
	})
	loads.Add(
		includes.IncludeRegistrationDetails || includes.IncludeRegistrationState,
		func() (keel.Loader, error) { return tr.RegistrationState(), nil },
	)
	loads.Add(includes.IncludeOwnerWorker, func() (keel.Loader, error) {
		return listquery.Load(tr.OwnerWorker().Query(), includes.OwnerWorkerColumns)
	})
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
		includes.IncludeFleetDetails || includes.IncludeFleetCode || includes.IncludeFleetManager,
		func() (keel.Loader, error) {
			nested := slices.Concat(
				listquery.When(includes.IncludeFleetManager, fc.Manager()),
				listquery.When(details, fc.BusinessUnit(), fc.Organization()),
			)
			return listquery.Load(fc.Query(), includes.FleetCodeColumns, nested...)
		},
	)

	return loads.Result()
}

const lastKnownLocationFilter = "lastKnownLocationId"

func withLastKnownLocation(
	q *keel.Query[*trailer.Trailer],
	includes *trailerports.TrailerRelationIncludes,
) *keel.Query[*trailer.Trailer] {
	if !includes.IncludeLastKnownLocation {
		return q
	}

	tr := trailerdb.Trailers
	ec := lastknownlocation.Continuity()
	location := lastknownlocation.Of(ec, lastknownlocation.Tenant{
		OrganizationID: tr.OrganizationID,
		BusinessUnitID: tr.BusinessUnitID,
	})
	current := ec.Pick(ec.CurrentLocationID).
		Where(lastknownlocation.Current(ec, tr.ID, equipmentcontinuity.EquipmentTypeTrailer)...).
		Limit(1)

	return q.LeftJoinLateral(current).
		Compute(tr.LastKnownLocationID, location.LocationID).
		Compute(tr.LastKnownLocationName, location.Name).
		Compute(tr.LastKnownLocationLatitude, location.Latitude).
		Compute(tr.LastKnownLocationLongitude, location.Longitude)
}
