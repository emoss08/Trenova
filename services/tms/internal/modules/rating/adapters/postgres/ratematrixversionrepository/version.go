package ratematrixversionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion/ratematrixversiondb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
	maxServedTables     = 1000
)

var versionErrors = dberror.Mapping{Entity: "RateMatrixVersion"}

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
	version *ratematrixversion.RateMatrixVersion,
) (*ratematrixversion.RateMatrixVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*ratematrixversion.RateMatrixVersion, error) {
			if err := ratematrixversiondb.RateMatrixVersions.Create(
				ctx,
				r.db.Keel(ctx),
				version,
			); err != nil {
				r.l.Error("failed to create rate matrix version",
					zap.String("matrixId", version.RateMatrixID.String()),
					zap.Int64("versionNumber", version.VersionNumber),
					zap.Error(err),
				)
				return nil, versionErrors.Map(err, version.ID.String())
			}

			return version, nil
		},
	)
}

func (r *repository) Get(
	ctx context.Context,
	req *ratematrixports.GetVersionRequest,
) (*ratematrixversion.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ratematrixversion.RateMatrixVersion, error) {
			v := ratematrixversiondb.RateMatrixVersions
			entity, err := v.Where(
				v.RateMatrixID.Eq(req.RateMatrixID),
				v.VersionNumber.Eq(req.VersionNumber),
			).Load(v.PublishedBy()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, versionErrors.Map(err, req.RateMatrixID.String())
			}
			if err = r.attachDimensions(ctx, entity); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *ratematrixports.ListVersionsRequest,
) ([]*ratematrixversion.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrixversion.RateMatrixVersion, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultHistoryLimit
			}
			limit = min(limit, maxHistoryLimit)

			v := ratematrixversiondb.RateMatrixVersions
			versions, err := v.Where(v.RateMatrixID.Eq(req.RateMatrixID)).
				Load(v.PublishedBy()).
				Order(v.VersionNumber.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list rate matrix versions", zap.Error(err))
				return nil, err
			}
			if err = r.attachDimensions(ctx, versions...); err != nil {
				return nil, err
			}

			return versions, nil
		},
	)
}

func (r *repository) GetByKeys(
	ctx context.Context,
	req *ratematrixports.GetVersionsByKeysRequest,
) ([]*ratematrixversion.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrixversion.RateMatrixVersion, error) {
			keys := uniqueKeys(req.Keys)
			if len(keys) == 0 {
				return []*ratematrixversion.RateMatrixVersion{}, nil
			}

			versions, err := versionsByKeysQuery(keys).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get rate matrix versions by keys", zap.Error(err))
				return nil, err
			}
			if err = r.attachDimensions(ctx, versions...); err != nil {
				return nil, err
			}

			return versions, nil
		},
	)
}

func versionsByKeysQuery(
	keys []ratematrixversion.VersionKey,
) *keel.Query[*ratematrixversion.RateMatrixVersion] {
	v := ratematrixversiondb.RateMatrixVersions
	wanted := versionKeys(keys)

	return v.With(wanted).
		Join(wanted, matchesKey(wanted, v.RateMatrixID.Col, v.VersionNumber.Col)...).
		Limit(len(keys))
}

func servedMatrices() []keel.Pred {
	rm := ratematrixdb.RateMatrixes

	return []keel.Pred{
		rm.PublishedVersionNumber.Gt(0),
		rm.Status.Ne(formulatemplate.StatusInactive),
	}
}

func servedQuery() *keel.Query[*ratematrixversion.RateMatrixVersion] {
	v := ratematrixversiondb.RateMatrixVersions
	rm := ratematrixdb.RateMatrixes

	return v.Join(
		rm,
		rm.ID.EqCol(v.RateMatrixID),
		rm.PublishedVersionNumber.EqCol(v.VersionNumber),
	).Where(servedMatrices()...).Order(v.Code.Asc()).Limit(maxServedTables)
}

func (r *repository) ListServed(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*ratematrixversion.RateMatrixVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ratematrixversion.RateMatrixVersion, error) {
			versions, err := servedQuery().List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list served rate matrix versions", zap.Error(err))
				return nil, err
			}
			if err = r.attachDimensions(ctx, versions...); err != nil {
				return nil, err
			}

			return versions, nil
		},
	)
}

func servedStampQuery() *keel.Query[*ratematrix.RateMatrix] {
	rm := ratematrixdb.RateMatrixes

	stamps := keel.StringAgg(
		rm.ID.Cat(keel.ConstText("@"), keel.CastText(rm.PublishedVersionNumber)),
		",",
	).OrderBy(rm.ID.Asc())

	return rm.Pick(keel.ConcatWS(":", keel.Count(), keel.StringOf(stamps).Coalesce("").MD5())).
		Where(servedMatrices()...)
}

func (r *repository) ServedStamp(
	ctx context.Context,
	_ pagination.TenantInfo,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		stamp, err := keel.IntoOne[string](ctx, r.db.Keel(ctx), servedStampQuery())
		if err != nil {
			r.l.Error("failed to read the served rate matrix stamp", zap.Error(err))
			return "", err
		}

		return stamp, nil
	})
}

func (r *repository) attachDimensions(
	ctx context.Context,
	versions ...*ratematrixversion.RateMatrixVersion,
) error {
	if len(versions) == 0 {
		return nil
	}

	keys := make([]ratematrixversion.VersionKey, 0, len(versions))
	for _, version := range versions {
		keys = append(keys, version.Key())
	}

	dimensions, err := dimensionsQuery(uniqueKeys(keys)).All(ctx, r.db.Keel(ctx))
	if err != nil {
		r.l.Error("failed to load rate matrix version dimensions", zap.Error(err))
		return err
	}

	byKey := make(
		map[ratematrixversion.VersionKey]*ratematrixversion.RateMatrixVersion,
		len(versions),
	)
	for _, version := range versions {
		version.Dimensions = make([]*ratematrix.RateMatrixDimension, 0, ratematrix.MaxDimensions)
		byKey[version.Key()] = version
	}
	for _, dimension := range dimensions {
		key := ratematrixversion.VersionKey{
			RateMatrixID:  dimension.RateMatrixID,
			VersionNumber: dimension.VersionNumber,
		}
		if version, ok := byKey[key]; ok {
			version.Dimensions = append(version.Dimensions, dimension)
		}
	}

	return nil
}

func dimensionsQuery(
	keys []ratematrixversion.VersionKey,
) *keel.Query[*ratematrix.RateMatrixDimension] {
	d := ratematrixdb.RateMatrixDimensions
	wanted := versionKeys(keys)

	return d.With(wanted).
		Join(wanted, matchesKey(wanted, d.RateMatrixID.Col, d.VersionNumber.Col)...).
		Order(d.RateMatrixID.Asc(), d.VersionNumber.Asc(), d.Position.Asc())
}

func versionKeys(keys []ratematrixversion.VersionKey) *keel.CTETable {
	matrixIDs := make([]string, 0, len(keys))
	numbers := make([]int64, 0, len(keys))
	for _, key := range keys {
		matrixIDs = append(matrixIDs, key.RateMatrixID.String())
		numbers = append(numbers, key.VersionNumber)
	}

	return keel.Unnest(
		"version_keys",
		[]string{"rate_matrix_id", "version_number"},
		matrixIDs,
		numbers,
	)
}

func matchesKey(wanted *keel.CTETable, matrixID, versionNumber keel.Col) []keel.Pred {
	return []keel.Pred{
		wanted.Col("rate_matrix_id").EqCol(matrixID),
		wanted.Col("version_number").EqCol(versionNumber),
	}
}

func uniqueKeys(keys []ratematrixversion.VersionKey) []ratematrixversion.VersionKey {
	seen := make(map[ratematrixversion.VersionKey]struct{}, len(keys))
	unique := make([]ratematrixversion.VersionKey, 0, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}

	return unique
}
