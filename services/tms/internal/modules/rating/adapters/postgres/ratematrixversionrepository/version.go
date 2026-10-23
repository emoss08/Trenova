package ratematrixversionrepository

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
	maxServedTables     = 1000
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

func New(p Params) ratematrixports.RateMatrixVersionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratematrix-version-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	version *ratematrix.RateMatrixVersion,
) (*ratematrix.RateMatrixVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrixVersion, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(version).Exec(ctx); err != nil {
			r.l.Error("failed to create rate matrix version",
				zap.String("matrixId", version.RateMatrixID.String()),
				zap.Int64("versionNumber", version.VersionNumber),
				zap.Error(err),
			)
			return nil, err
		}
		return version, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req *ratematrixports.GetVersionRequest,
) (*ratematrix.RateMatrixVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrixVersion, error) {
		cols := ratematrixcol.RateMatrixVersionColumns
		entity := new(ratematrix.RateMatrixVersion)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(buncolgen.Rel(ratematrixcol.RateMatrixVersionRelations.PublishedBy)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID).
					Where(cols.VersionNumber.Eq(), req.VersionNumber)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "RateMatrixVersion")
		}
		if err = r.attachDimensions(ctx, req.TenantInfo, entity); err != nil {
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req *ratematrixports.ListVersionsRequest,
) ([]*ratematrix.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrix.RateMatrixVersion, error) {
			cols := ratematrixcol.RateMatrixVersionColumns
			limit := req.Limit
			if limit <= 0 {
				limit = defaultHistoryLimit
			}
			limit = min(limit, maxHistoryLimit)

			versions := make([]*ratematrix.RateMatrixVersion, 0, 8)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&versions).
				Relation(buncolgen.Rel(ratematrixcol.RateMatrixVersionRelations.PublishedBy)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return ratematrixcol.RateMatrixVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
				}).
				Order(cols.VersionNumber.OrderDesc()).
				Limit(int64(limit)).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list rate matrix versions", zap.Error(err))
				return nil, err
			}
			if err = r.attachDimensions(ctx, req.TenantInfo, versions...); err != nil {
				return nil, err
			}
			return versions, nil
		},
	)
}

func (r *repository) GetByKeys(
	ctx context.Context,
	req *ratematrixports.GetVersionsByKeysRequest,
) ([]*ratematrix.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrix.RateMatrixVersion, error) {
			if len(req.Keys) == 0 {
				return []*ratematrix.RateMatrixVersion{}, nil
			}
			matrixIDs, numbers := splitKeys(req.Keys)
			cols := ratematrixcol.RateMatrixVersionColumns
			candidates := make([]*ratematrix.RateMatrixVersion, 0, len(req.Keys))
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&candidates).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return ratematrixcol.RateMatrixVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.RateMatrixID.In(), bun.List(matrixIDs)).
						Where(cols.VersionNumber.In(), bun.List(numbers))
				}).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to get rate matrix versions by keys", zap.Error(err))
				return nil, err
			}

			wanted := make(map[ratematrix.VersionKey]struct{}, len(req.Keys))
			for _, key := range req.Keys {
				wanted[key] = struct{}{}
			}
			versions := candidates[:0]
			for _, version := range candidates {
				if _, ok := wanted[version.Key()]; ok {
					versions = append(versions, version)
				}
			}
			if err = r.attachDimensions(ctx, req.TenantInfo, versions...); err != nil {
				return nil, err
			}
			return versions, nil
		},
	)
}

func (r *repository) ListServed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*ratematrix.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrix.RateMatrixVersion, error) {
			versionCols := ratematrixcol.RateMatrixVersionColumns
			matrixCols := ratematrixcol.RateMatrixColumns
			versions := make([]*ratematrix.RateMatrixVersion, 0, 16)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&versions).
				Join(
					"JOIN ? AS ? ON ? AND ? AND ? AND ?",
					bun.Ident(ratematrixcol.RateMatrixTable.Name),
					bun.Ident(ratematrixcol.RateMatrixTable.Alias),
					bun.Safe(matrixCols.ID.EqColumn(versionCols.RateMatrixID)),
					bun.Safe(matrixCols.OrganizationID.EqColumn(versionCols.OrganizationID)),
					bun.Safe(matrixCols.BusinessUnitID.EqColumn(versionCols.BusinessUnitID)),
					bun.Safe(matrixCols.PublishedVersionNumber.EqColumn(versionCols.VersionNumber)),
				).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return ratematrixcol.RateMatrixScopeTenant(sq, tenantInfo).
						Where(matrixCols.PublishedVersionNumber.Gt(), 0).
						Where(matrixCols.Status.Ne(), formulatemplate.StatusInactive)
				}).
				Order(versionCols.Code.OrderAsc()).
				Limit(maxServedTables).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list served rate matrix versions", zap.Error(err))
				return nil, err
			}
			if err = r.attachDimensions(ctx, tenantInfo, versions...); err != nil {
				return nil, err
			}
			return versions, nil
		},
	)
}

func (r *repository) ServedStamp(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		cols := ratematrixcol.RateMatrixColumns
		var stamp string
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratematrix.RateMatrix)(nil)).
			ColumnExpr(buncolgen.Expr(
				"concat_ws(':', count(*), md5(coalesce(string_agg({0} || '@' || {1}, ',' ORDER BY {0}), '')))",
				cols.ID,
				cols.PublishedVersionNumber,
			)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixScopeTenant(sq, tenantInfo).
					Where(cols.PublishedVersionNumber.Gt(), 0).
					Where(cols.Status.Ne(), formulatemplate.StatusInactive)
			}).
			Scan(ctx, &stamp)
		if err != nil {
			r.l.Error("failed to read the served rate matrix stamp", zap.Error(err))
			return "", err
		}
		return stamp, nil
	})
}

func (r *repository) attachDimensions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	versions ...*ratematrix.RateMatrixVersion,
) error {
	if len(versions) == 0 {
		return nil
	}
	keys := make([]ratematrix.VersionKey, 0, len(versions))
	for _, version := range versions {
		keys = append(keys, version.Key())
	}
	matrixIDs, numbers := splitKeys(keys)

	cols := ratematrixcol.RateMatrixDimensionColumns
	dimensions := make([]*ratematrix.RateMatrixDimension, 0, len(versions)*2)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&dimensions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return ratematrixcol.RateMatrixDimensionScopeTenant(sq, tenantInfo).
				Where(cols.RateMatrixID.In(), bun.List(matrixIDs)).
				Where(cols.VersionNumber.In(), bun.List(numbers))
		}).
		Order(cols.RateMatrixID.OrderAsc(), cols.VersionNumber.OrderAsc(), cols.Position.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to load rate matrix version dimensions", zap.Error(err))
		return err
	}

	byKey := make(map[ratematrix.VersionKey]*ratematrix.RateMatrixVersion, len(versions))
	for _, version := range versions {
		version.Dimensions = make([]*ratematrix.RateMatrixDimension, 0, ratematrix.MaxDimensions)
		byKey[version.Key()] = version
	}
	for _, dimension := range dimensions {
		key := ratematrix.VersionKey{
			RateMatrixID:  dimension.RateMatrixID,
			VersionNumber: dimension.VersionNumber,
		}
		if version, ok := byKey[key]; ok {
			version.Dimensions = append(version.Dimensions, dimension)
		}
	}
	return nil
}

func splitKeys(keys []ratematrix.VersionKey) ([]pulid.ID, []int64) {
	matrixIDs := make([]pulid.ID, 0, len(keys))
	numbers := make([]int64, 0, len(keys))
	for _, key := range keys {
		if !slices.Contains(matrixIDs, key.RateMatrixID) {
			matrixIDs = append(matrixIDs, key.RateMatrixID)
		}
		if !slices.Contains(numbers, key.VersionNumber) {
			numbers = append(numbers, key.VersionNumber)
		}
	}
	return matrixIDs, numbers
}
