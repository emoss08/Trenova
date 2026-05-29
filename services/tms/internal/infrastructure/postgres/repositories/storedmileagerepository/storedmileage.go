package storedmileagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.StoredMileageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.stored-mileage-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListStoredMileageRequest,
) (*pagination.ListResult[*storedmileage.StoredMileage], error) {
	entities := make([]*storedmileage.StoredMileage, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.StoredMileageColumns
	total, err := r.db.DBForContext(ctx).NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			q = querybuilder.ApplyFilters(
				q,
				buncolgen.StoredMileageTable.Alias,
				req.Filter,
				(*storedmileage.StoredMileage)(nil),
			)
			return q.Apply(buncolgen.StoredMileageApplyTenant(req.Filter.TenantInfo)).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset()).
				Order(cols.LastCalculatedAt.OrderDesc())
		}).
		ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list stored mileages", zap.Error(err))
		return nil, err
	}
	return &pagination.ListResult[*storedmileage.StoredMileage]{Items: entities, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetStoredMileageByIDRequest,
) (*storedmileage.StoredMileage, error) {
	entity := new(storedmileage.StoredMileage)
	cols := buncolgen.StoredMileageColumns
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.StoredMileageScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "StoredMileage")
	}
	return entity, nil
}

func (r *repository) Lookup(
	ctx context.Context,
	req repositories.StoredMileageLookupRequest,
) (*storedmileage.StoredMileage, error) {
	entity := new(storedmileage.StoredMileage)
	cols := buncolgen.StoredMileageColumns
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.StoredMileageScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), storedmileage.StatusActive).
				Where(cols.RouteHash.Eq(), req.RouteHash).
				Where(cols.DistanceUnits.Eq(), req.DistanceUnits).
				Where(cols.RoutingType.Eq(), req.RoutingType).
				Where(cols.Method.Eq(), req.Method).
				Where(cols.DistanceProfileID.Eq(), req.DistanceProfileID).
				Where(cols.HazmatSignature.Eq(), req.HazmatSignature)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "StoredMileage")
	}
	return entity, nil
}

func (r *repository) BulkUpsert(ctx context.Context, entities []*storedmileage.StoredMileage) error {
	if len(entities) == 0 {
		return nil
	}
	entities = dedupeUpsertEntities(entities)
	if len(entities) == 0 {
		return nil
	}
	for _, entity := range entities {
		entity.ApplyDefaults()
	}
	cols := buncolgen.StoredMileageColumns
	_, err := r.db.DBForContext(ctx).NewInsert().
		Model(&entities).
		Column(buncolgen.StoredMileageInsertableColumns...).
		On(storedMileageActiveUpsertConflictClause()).
		Set(cols.Distance.SetExcluded()).
		Set(cols.Provider.SetExcluded()).
		Set(cols.Source.SetExcluded()).
		Set(cols.DataVersion.SetExcluded()).
		Set(cols.DistanceProfileName.SetExcluded()).
		Set(cols.ProviderMetadata.SetExcluded()).
		Set(cols.LastCalculatedAt.SetExcluded()).
		Set(cols.Version.SetExpr(cols.Version.Qualified() + " + 1")).
		Set(cols.UpdatedAt.SetExcluded()).
		Exec(ctx)
	return err
}

func storedMileageActiveUpsertConflictClause() string {
	cols := buncolgen.StoredMileageColumns
	conflictColumns := []string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.RouteHash.Bare(),
		cols.DistanceUnits.Bare(),
		cols.RoutingType.Bare(),
		cols.Method.Bare(),
		cols.DistanceProfileID.Bare(),
		cols.HazmatSignature.Bare(),
	}
	return "CONFLICT (" + strings.Join(conflictColumns, ", ") + ") WHERE " +
		cols.Status.Bare() + " = 'Active' DO UPDATE"
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
	tenantInfo pagination.TenantInfo,
) error {
	now := timeutils.NowUnix()
	cols := buncolgen.StoredMileageColumns
	_, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*storedmileage.StoredMileage)(nil)).
		Set(cols.HitCount.Inc(1)).
		Set(cols.LastUsedAt.Set(), now).
		Set(cols.UpdatedAt.Set(), now).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.StoredMileageScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.Eq(), id)
		}).
		Exec(ctx)
	return err
}

func (r *repository) Deactivate(
	ctx context.Context,
	req repositories.DeleteStoredMileageRequest,
) error {
	now := timeutils.NowUnix()
	cols := buncolgen.StoredMileageColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*storedmileage.StoredMileage)(nil)).
		Set(cols.Status.Set(), storedmileage.StatusInactive).
		Set(cols.UpdatedAt.Set(), now).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.StoredMileageScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "StoredMileage", req.ID.String())
}
