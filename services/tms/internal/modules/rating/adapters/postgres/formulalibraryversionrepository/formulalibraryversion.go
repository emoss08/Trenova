package formulalibraryversionrepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/formulalibrarycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit  = 100
	maxHistoryLimit      = 500
	maxApprovedLibraries = 500
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
			if _, err := r.db.DBForContext(ctx).NewInsert().Model(version).Exec(ctx); err != nil {
				r.l.Error("failed to create formula library version",
					zap.String("libraryID", version.LibraryID.String()),
					zap.Int64("versionNumber", version.VersionNumber),
					zap.Error(err),
				)
				return nil, err
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
			cols := formulalibrarycol.FormulaLibraryVersionColumns
			entity := new(formulalibrary.FormulaLibraryVersion)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.LibraryID.Eq(), req.LibraryID).
						Where(cols.VersionNumber.Eq(), req.VersionNumber)
				}).
				Scan(ctx)
			if err != nil {
				return nil, dberror.HandleNotFoundError(err, "FormulaLibraryVersion")
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
			cols := formulalibrarycol.FormulaLibraryVersionColumns
			limit := req.Limit
			if limit <= 0 {
				limit = defaultHistoryLimit
			}
			limit = min(limit, maxHistoryLimit)

			versions := make([]*formulalibrary.FormulaLibraryVersion, 0, 8)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&versions).
				Relation(buncolgen.Rel(formulalibrarycol.FormulaLibraryVersionRelations.CreatedBy)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.LibraryID.Eq(), req.LibraryID)
				}).
				Order(cols.VersionNumber.OrderDesc()).
				Limit(int64(limit)).
				Scan(ctx)
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
			cols := formulalibrarycol.FormulaLibraryVersionColumns
			versions := make([]*formulalibrary.FormulaLibraryVersion, 0, 2)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&versions).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.LibraryID.Eq(), req.LibraryID).
						Where(cols.VersionNumber.In(), bun.List([]int64{req.FromVersion, req.ToVersion}))
				}).
				Order(cols.VersionNumber.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to read formula library versions", zap.Error(err))
				return nil, err
			}
			return versions, nil
		},
	)
}

func (r *repository) latestApproved(
	ctx context.Context,
	apply func(*bun.SelectQuery) *bun.SelectQuery,
	limit int,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	versions := formulalibrarycol.FormulaLibraryVersionColumns
	libraries := formulalibrarycol.FormulaLibraryColumns

	out := make([]*formulalibrary.FormulaLibraryVersion, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&out).
		DistinctOn(versions.LibraryID.Qualified()).
		Join(
			"JOIN ? AS ? ON ? AND ? AND ?",
			bun.Ident(formulalibrarycol.FormulaLibraryTable.Name),
			bun.Ident(formulalibrarycol.FormulaLibraryTable.Alias),
			bun.Safe(libraries.ID.EqColumn(versions.LibraryID)),
			bun.Safe(libraries.OrganizationID.EqColumn(versions.OrganizationID)),
			bun.Safe(libraries.BusinessUnitID.EqColumn(versions.BusinessUnitID)),
		).
		Where(versions.Status.Eq(), formulatemplate.StatusActive).
		Where(libraries.Status.NotEq(), formulatemplate.StatusInactive).
		Apply(apply).
		OrderExpr(versions.LibraryID.OrderAsc()).
		OrderExpr(versions.VersionNumber.OrderDesc()).
		Limit(int64(limit)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return out, nil
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

			libraries := formulalibrarycol.FormulaLibraryColumns
			out, err := r.latestApproved(ctx, func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulalibrarycol.FormulaLibraryScopeTenant(sq, req.TenantInfo).
					Where(libraries.Name.Expr("lower({})")+" IN (?)", bun.List(lowered))
			}, len(lowered))
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
			out, err := r.latestApproved(ctx, func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulalibrarycol.FormulaLibraryScopeTenant(sq, req.TenantInfo)
			}, limit)
			if err != nil {
				r.l.Error("failed to list approved formula libraries", zap.Error(err))
				return nil, err
			}
			return out, nil
		},
	)
}

func (r *repository) GetByPins(
	ctx context.Context,
	req *formulalibraryports.GetVersionsByPinsRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error) {
			out := make([]*formulalibrary.FormulaLibraryVersion, 0, len(req.Pins))
			if len(req.Pins) == 0 {
				return out, nil
			}

			cols := formulalibrarycol.FormulaLibraryVersionColumns
			pairs := make([]any, 0, len(req.Pins))
			placeholders := make([]string, 0, len(req.Pins))
			for _, pin := range req.Pins {
				pairs = append(pairs, pin.LibraryID, pin.VersionNumber)
				placeholders = append(placeholders, "(?, ?)")
			}

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&out).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
						Where(
							buncolgen.Expr("({0}, {1})", cols.LibraryID, cols.VersionNumber)+
								" IN ("+strings.Join(placeholders, ", ")+")",
							pairs...,
						)
				}).
				Order(cols.LibraryID.OrderAsc()).
				Scan(ctx)
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
		cols := formulalibrarycol.FormulaLibraryVersionColumns
		ids := make([]pulid.ID, 0, 4)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*formulalibrary.FormulaLibraryVersion)(nil)).
			Distinct().
			Column(cols.CreatedByID.String()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.LibraryID.Eq(), req.LibraryID).
					Where(cols.VersionNumber.Gt(), req.AfterVersion)
			}).
			Scan(ctx, &ids)
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
			cols := formulalibrarycol.FormulaLibraryVersionColumns
			version := new(formulalibrary.FormulaLibraryVersion)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(version).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryVersionScopeTenant(sq, req.TenantInfo).
						Where(cols.LibraryID.Eq(), req.LibraryID).
						Where(cols.Status.Eq(), req.Status)
				}).
				Order(cols.VersionNumber.OrderDesc()).
				Limit(1).
				Scan(ctx)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, nil
				}
				r.l.Error("failed to read the latest formula library version", zap.Error(err))
				return nil, err
			}
			return version, nil
		},
	)
}
