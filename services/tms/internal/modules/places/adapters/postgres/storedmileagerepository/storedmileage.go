package storedmileagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage/storedmileagedb"
	"github.com/emoss08/trenova/internal/modules/places/ports/storedmileageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	bulkUpsertScopeReason = "store a batch of mileage lookups that can span organizations"
	activePolicySQL       = "status = 'Active'"
)

var storedMileageErrors = dberror.Mapping{Entity: "StoredMileage"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) storedmileageports.StoredMileageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.stored-mileage-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *storedmileageports.ListStoredMileageRequest,
) (*pagination.ListResult[*storedmileage.StoredMileage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*storedmileage.StoredMileage], error) {
			sm := storedmileagedb.StoredMileages
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				sm.Query(),
				&listquery.Request[storedmileage.StoredMileage]{
					Filter: req.Filter,
					Then:   []keel.Order{sm.LastCalculatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list stored mileages", zap.Error(err))
				return nil, err
			}
			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *storedmileageports.ListStoredMileageConnectionRequest,
) (*pagination.CursorListResult[*storedmileage.StoredMileage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*storedmileage.StoredMileage], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				storedmileagedb.StoredMileages.Query(),
				&listquery.Request[storedmileage.StoredMileage]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.StoredMileageColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list stored mileages", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req storedmileageports.GetStoredMileageByIDRequest,
) (*storedmileage.StoredMileage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*storedmileage.StoredMileage, error) {
		entity, err := storedmileagedb.StoredMileages.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, storedMileageErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) Lookup(
	ctx context.Context,
	req storedmileageports.StoredMileageLookupRequest,
) (*storedmileage.StoredMileage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*storedmileage.StoredMileage, error) {
		sm := storedmileagedb.StoredMileages
		entity, err := sm.Where(
			sm.Status.Eq(storedmileage.StatusActive),
			sm.RouteHash.Eq(req.RouteHash),
			sm.DistanceUnits.Eq(req.DistanceUnits),
			sm.RoutingType.Eq(req.RoutingType),
			sm.Method.Eq(req.Method),
			sm.DistanceProfileID.Eq(req.DistanceProfileID),
			sm.HazmatSignature.Eq(req.HazmatSignature),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, storedMileageErrors.Map(err, req.RouteHash)
		}
		return entity, nil
	})
}

func (r *repository) BulkUpsert(
	ctx context.Context,
	entities []*storedmileage.StoredMileage,
) error {
	ctx = dbscope.WithSystem(ctx, bulkUpsertScopeReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}
		entities = dedupeUpsertEntities(entities)
		if len(entities) == 0 {
			return nil
		}

		sm := storedmileagedb.StoredMileages
		_, err := sm.Upsert(
			ctx,
			r.db.Keel(ctx),
			entities,
			keel.OnConflict(
				sm.OrganizationID,
				sm.BusinessUnitID,
				sm.RouteHash,
				sm.DistanceUnits,
				sm.RoutingType,
				sm.Method,
				sm.DistanceProfileID,
				sm.HazmatSignature,
			).
				Where(keel.Raw(activePolicySQL)).
				Update(
					sm.Distance,
					sm.Provider,
					sm.Source,
					sm.DataVersion,
					sm.DistanceProfileName,
					sm.ProviderMetadata,
					sm.LastCalculatedAt,
				).
				Set(sm.JurisdictionDistances.ToExpr(keel.DynOf(sm.JurisdictionDistances.Excluded()).Coalesce(sm.JurisdictionDistances))),
		)
		return err
	})
}

func dedupeUpsertEntities(
	entities []*storedmileage.StoredMileage,
) []*storedmileage.StoredMileage {
	byKey := make(map[storedMileageUpsertKey]*storedmileage.StoredMileage, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		entity.ApplyDefaults()
		key := upsertKey(entity)
		existing, ok := byKey[key]
		if !ok || entity.LastCalculatedAt >= existing.LastCalculatedAt {
			byKey[key] = entity
		}
	}
	result := make([]*storedmileage.StoredMileage, 0, len(byKey))
	for _, entity := range byKey {
		result = append(result, entity)
	}
	return result
}

type storedMileageUpsertKey struct {
	organizationID    pulid.ID
	businessUnitID    pulid.ID
	routeHash         string
	distanceUnits     string
	routingType       string
	method            string
	distanceProfileID pulid.ID
	hazmatSignature   string
}

func upsertKey(entity *storedmileage.StoredMileage) storedMileageUpsertKey {
	return storedMileageUpsertKey{
		organizationID:    entity.OrganizationID,
		businessUnitID:    entity.BusinessUnitID,
		routeHash:         entity.RouteHash,
		distanceUnits:     entity.DistanceUnits,
		routingType:       entity.RoutingType,
		method:            entity.Method,
		distanceProfileID: entity.DistanceProfileID,
		hazmatSignature:   entity.HazmatSignature,
	}
}

func (r *repository) IncrementHit(
	ctx context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		now := timeutils.NowUnix()
		sm := storedmileagedb.StoredMileages
		_, err := sm.Where(sm.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			sm.HitCount.Add(1),
			sm.LastUsedAt.To(now),
			sm.UpdatedAt.To(now),
			sm.Version.ToExpr(sm.Version),
		)
		return err
	})
}

func (r *repository) Deactivate(
	ctx context.Context,
	req storedmileageports.DeleteStoredMileageRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sm := storedmileagedb.StoredMileages
		affected, err := sm.Where(sm.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			sm.Status.To(storedmileage.StatusInactive),
			sm.UpdatedAt.To(timeutils.NowUnix()),
			sm.Version.ToExpr(sm.Version),
		)
		if err != nil {
			return storedMileageErrors.Map(err, req.ID.String())
		}
		return dberror.CheckAffected(affected, "StoredMileage", req.ID.String())
	})
}
