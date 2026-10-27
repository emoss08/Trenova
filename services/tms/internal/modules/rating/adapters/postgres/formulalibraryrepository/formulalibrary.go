package formulalibraryrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxNamesLike = 200

var libraryErrors = dberror.Mapping{Entity: "FormulaLibrary"}

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

func mapLibraryError(err error, entity *formulalibrary.FormulaLibrary) error {
	if dberror.IsUniqueConstraintViolation(err) {
		return duplicateLibraryName(entity.Name)
	}

	return libraryErrors.Map(err, entity.ID.String())
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		if err := formulalibrarydb.FormulaLibraries.Create(
			ctx,
			r.db.Keel(ctx),
			entity,
		); err != nil {
			r.l.Error("failed to create formula library",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, mapLibraryError(err, entity)
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		if err := formulalibrarydb.FormulaLibraries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update formula library",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, mapLibraryError(err, entity)
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *formulalibraryports.GetLibraryByIDRequest,
) (*formulalibrary.FormulaLibrary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*formulalibrary.FormulaLibrary, error) {
		entity, err := formulalibrarydb.FormulaLibraries.Get(ctx, r.db.Keel(ctx), req.LibraryID)
		if err != nil {
			return nil, libraryErrors.Map(err, req.LibraryID.String())
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
			if len(req.LibraryIDs) == 0 {
				return []*formulalibrary.FormulaLibrary{}, nil
			}

			fl := formulalibrarydb.FormulaLibraries
			entities, err := fl.Where(fl.ID.In(req.LibraryIDs...)).
				Limit(len(req.LibraryIDs)).
				List(ctx, r.db.Keel(ctx))
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
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				formulalibrarydb.FormulaLibraries.Query(),
				&listquery.Request[formulalibrary.FormulaLibrary]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FormulaLibraryColumns,
				},
			)
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
	_ pagination.TenantInfo,
	prefix string,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		names, err := keel.Into[string](ctx, r.db.Keel(ctx), namesLikeQuery(prefix))
		if err != nil {
			r.l.Error("failed to read formula library names", zap.Error(err))
			return nil, err
		}

		return names, nil
	})
}

func namesLikeQuery(prefix string) *keel.Query[*formulalibrary.FormulaLibrary] {
	fl := formulalibrarydb.FormulaLibraries

	return fl.Pick(fl.Name).Where(fl.Name.HasPrefixFold(prefix)).Limit(maxNamesLike)
}
