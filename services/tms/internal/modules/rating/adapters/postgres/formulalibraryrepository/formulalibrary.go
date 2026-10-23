package formulalibraryrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/formulalibrarycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxNamesLike = 200

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) formulalibraryports.FormulaLibraryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-library-repository"),
	}
}

func duplicateLibraryName(name string) error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		"A formula library named \"{0}\" already exists",
		name,
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateLibraryName(entity.Name)
			}
			r.l.Error("failed to create formula library",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		cols := formulalibrarycol.FormulaLibraryColumns
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Value(cols.Description.String(), "?", entity.Description).
			Value(cols.ReviewComment.String(), "?", entity.ReviewComment).
			Value(cols.TestCount.String(), "?", entity.TestCount).
			Value(cols.SubmittedByID.String(), "?", entity.SubmittedByID).
			Value(cols.SubmittedAt.String(), "?", entity.SubmittedAt).
			Value(cols.ApprovedByID.String(), "?", entity.ApprovedByID).
			Value(cols.ApprovedAt.String(), "?", entity.ApprovedAt).
			Returning("*").
			Exec(ctx)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateLibraryName(entity.Name)
			}
			r.l.Error("failed to update formula library",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			results,
			"FormulaLibrary",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *formulalibraryports.GetLibraryByIDRequest,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		entity := new(formulalibrary.FormulaLibrary)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulalibrarycol.FormulaLibraryScopeTenant(sq, req.TenantInfo).
					Where(formulalibrarycol.FormulaLibraryColumns.ID.Eq(), req.LibraryID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "FormulaLibrary")
		}
		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *formulalibraryports.GetLibrariesByIDsRequest,
) ([]*formulalibrary.FormulaLibrary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibrary, error) {
			entities := make([]*formulalibrary.FormulaLibrary, 0, len(req.LibraryIDs))
			if len(req.LibraryIDs) == 0 {
				return entities, nil
			}
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryScopeTenant(sq, req.TenantInfo).
						Where(formulalibrarycol.FormulaLibraryColumns.ID.In(), bun.List(req.LibraryIDs))
				}).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to get formula libraries", zap.Error(err))
				return nil, err
			}
			return entities, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *formulalibraryports.ListLibraryConnectionRequest,
) (*pagination.CursorListResult[*formulalibrary.FormulaLibrary], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*formulalibrary.FormulaLibrary], error) {
			dba := r.db.DBForContext(ctx)
			alias := formulalibrarycol.FormulaLibraryTable.Alias

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*formulalibrary.FormulaLibrary)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return querybuilder.ApplyFiltersWithoutSort(
							sq,
							alias,
							req.Filter,
							(*formulalibrary.FormulaLibrary)(nil),
						)
					}).
					Count(ctx)
				if err != nil {
					r.l.Error("failed to count formula libraries", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*formulalibrary.FormulaLibrary]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*formulalibrary.FormulaLibrary) *bun.SelectQuery {
						q := dba.NewSelect().Model(entities)
						if len(req.FormulaLibraryColumns) > 0 {
							q = q.Column(req.FormulaLibraryColumns...)
						}
						return q
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							sq,
							alias,
							req.Filter,
							req.Cursor,
							(*formulalibrary.FormulaLibrary)(nil),
						)
					},
				})
			if err != nil {
				r.l.Error("failed to list formula libraries", zap.Error(err))
				return nil, err
			}
			return result, nil
		},
	)
}

func (r *repository) NamesLike(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	prefix string,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		cols := formulalibrarycol.FormulaLibraryColumns
		names := make([]string, 0, 4)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*formulalibrary.FormulaLibrary)(nil)).
			Column(cols.Name.String()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulalibrarycol.FormulaLibraryScopeTenant(sq, tenantInfo).
					Where(cols.Name.Expr("lower({})")+" LIKE lower(?)", prefix+"%")
			}).
			Limit(int64(maxNamesLike)).
			Scan(ctx, &names)
		if err != nil {
			r.l.Error("failed to read formula library names", zap.Error(err))
			return nil, err
		}
		return names, nil
	})
}
