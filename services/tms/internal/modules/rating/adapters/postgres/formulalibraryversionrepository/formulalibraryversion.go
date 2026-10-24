package formulalibraryversionrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit  = 100
	maxHistoryLimit      = 500
	maxApprovedLibraries = 500
)

var versionErrors = dberror.Mapping{Entity: "FormulaLibraryVersion"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) formulalibraryports.FormulaLibraryVersionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-library-version-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	version *formulalibrary.FormulaLibraryVersion,
) (*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryVersion, error) {
			err := formulalibrarydb.FormulaLibraryVersions.Create(ctx, r.db.Keel(ctx), version)
			if err != nil {
				r.l.Error("failed to create formula library version",
					zap.String("libraryID", version.LibraryID.String()),
					zap.Int64("versionNumber", version.VersionNumber),
					zap.Error(err),
				)
				return nil, versionErrors.Map(err, version.ID.String())
			}

			return version, nil
		},
	)
}

func (r *repository) GetByLibraryAndVersion(
	ctx context.Context,
	req *formulalibraryports.GetLibraryVersionRequest,
) (*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryVersion, error) {
			v := formulalibrarydb.FormulaLibraryVersions
			entity, err := v.Where(
				v.LibraryID.Eq(req.LibraryID),
				v.VersionNumber.Eq(req.VersionNumber),
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, versionErrors.Map(err, req.LibraryID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *formulalibraryports.ListLibraryVersionsRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultHistoryLimit
			}
			limit = min(limit, maxHistoryLimit)

			v := formulalibrarydb.FormulaLibraryVersions
			versions, err := v.Where(v.LibraryID.Eq(req.LibraryID)).
				Load(v.CreatedBy()).
				Order(v.VersionNumber.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list formula library versions", zap.Error(err))
				return nil, err
			}

			return versions, nil
		},
	)
}

func (r *repository) GetVersionRange(
	ctx context.Context,
	req *formulalibraryports.GetLibraryVersionRangeRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			v := formulalibrarydb.FormulaLibraryVersions
			versions, err := v.Where(
				v.LibraryID.Eq(req.LibraryID),
				v.VersionNumber.In(req.FromVersion, req.ToVersion),
			).Order(v.VersionNumber.Asc()).Limit(2).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to read formula library versions", zap.Error(err))
				return nil, err
			}

			return versions, nil
		},
	)
}

func latestApprovedQuery(
	limit int,
	filters ...keel.Pred,
) *keel.Query[formulalibrary.FormulaLibraryVersion] {
	v := formulalibrarydb.FormulaLibraryVersions
	fl := formulalibrarydb.FormulaLibraries

	return v.DistinctOn(v.LibraryID).
		Join(fl, fl.ID.EqCol(v.LibraryID)).
		Where(v.Status.Eq(formulatemplate.StatusActive), fl.Status.Ne(formulatemplate.StatusInactive)).
		Where(filters...).
		Order(v.LibraryID.Asc(), v.VersionNumber.Desc()).
		Limit(limit)
}

func (r *repository) LatestApprovedByNames(
	ctx context.Context,
	req *formulalibraryports.LatestApprovedByNamesRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			if len(req.Names) == 0 {
				return []*formulalibrary.FormulaLibraryVersion{}, nil
			}
			lowered := make([]string, 0, len(req.Names))
			for _, name := range req.Names {
				lowered = append(lowered, strings.ToLower(name))
			}

			fl := formulalibrarydb.FormulaLibraries
			out, err := latestApprovedQuery(len(lowered), fl.Name.Lower().In(lowered...)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to resolve formula libraries by name", zap.Error(err))
				return nil, err
			}

			return out, nil
		},
	)
}

func (r *repository) ListLatestApproved(
	ctx context.Context,
	req *formulalibraryports.ListApprovedLibrariesRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			limit := req.Limit
			if limit <= 0 || limit > maxApprovedLibraries {
				limit = maxApprovedLibraries
			}

			out, err := latestApprovedQuery(limit).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list approved formula libraries", zap.Error(err))
				return nil, err
			}

			return out, nil
		},
	)
}

func pinsQuery(pins []tarifftypes.Import) *keel.Query[formulalibrary.FormulaLibraryVersion] {
	ids := make([]string, 0, len(pins))
	numbers := make([]int64, 0, len(pins))
	for _, pin := range pins {
		ids = append(ids, pin.LibraryID.String())
		numbers = append(numbers, pin.VersionNumber)
	}

	v := formulalibrarydb.FormulaLibraryVersions
	pinned := keel.Unnest("pins", []string{"library_id", "version_number"}, ids, numbers)

	return v.With(pinned).
		Join(
			pinned,
			pinned.Col("library_id").EqCol(v.LibraryID),
			pinned.Col("version_number").EqCol(v.VersionNumber),
		).
		Order(v.LibraryID.Asc())
}

func (r *repository) GetByPins(
	ctx context.Context,
	req *formulalibraryports.GetVersionsByPinsRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			if len(req.Pins) == 0 {
				return []*formulalibrary.FormulaLibraryVersion{}, nil
			}

			out, err := pinsQuery(req.Pins).Limit(len(req.Pins)).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to load pinned formula library versions", zap.Error(err))
				return nil, err
			}

			return out, nil
		},
	)
}

func (r *repository) ListAuthorIDsSince(
	ctx context.Context,
	req *formulalibraryports.ListLibraryAuthorsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		v := formulalibrarydb.FormulaLibraryVersions
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), v.Group(v.CreatedByID).
			Pick(v.CreatedByID).
			Where(v.LibraryID.Eq(req.LibraryID), v.VersionNumber.Gt(req.AfterVersion)))
		if err != nil {
			r.l.Error("failed to list formula library authors", zap.Error(err))
			return nil, err
		}

		return ids, nil
	})
}

func (r *repository) GetLatestByStatus(
	ctx context.Context,
	req *formulalibraryports.GetLatestLibraryVersionByStatusRequest,
) (*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryVersion, error) {
			v := formulalibrarydb.FormulaLibraryVersions
			version, err := v.Where(v.LibraryID.Eq(req.LibraryID), v.Status.Eq(req.Status)).
				Order(v.VersionNumber.Desc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil
				}
				r.l.Error("failed to read the latest formula library version", zap.Error(err))
				return nil, err
			}

			return version, nil
		},
	)
}
