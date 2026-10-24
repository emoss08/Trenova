package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval/extractionevaldb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

var casePagesErrors = dberror.Mapping{Entity: "ExtractionEvalCasePages"}

type casePagesRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCasePages(p Params) extractionevalports.ExtractionEvalCasePagesRepository {
	return &casePagesRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-case-pages-repository"),
	}
}

func (r *casePagesRepository) Upsert(
	ctx context.Context,
	entity *extractioneval.CasePages,
) (*extractioneval.CasePages, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractioneval.CasePages, error) {
		cp := extractionevaldb.CasePageses
		db := r.db.Keel(ctx)
		inserted, err := cp.Upsert(
			ctx,
			db,
			[]*extractioneval.CasePages{entity},
			keel.OnConflict(cp.OrganizationID, cp.BusinessUnitID, cp.CaseID).Update(
				cp.Pages,
				cp.PageCount,
				cp.InputHash,
				cp.ConverterVersion,
				cp.FrozenByID,
				cp.FrozenAt,
			),
		)
		if err != nil {
			r.l.Error("failed to save extraction eval case pages", zap.Error(err))

			return nil, fmt.Errorf("save extraction eval case pages: %w", err)
		}
		if len(inserted) == 1 && inserted[0] {
			return entity, nil
		}

		stored, err := cp.Where(cp.CaseID.Eq(entity.CaseID)).One(ctx, db)
		if err != nil {
			return nil, casePagesErrors.Map(err, entity.CaseID.String())
		}
		*entity = *stored

		return entity, nil
	})
}

func (r *casePagesRepository) Get(
	ctx context.Context,
	req *extractionevalports.GetExtractionEvalCasePagesRequest,
) (*extractioneval.CasePages, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractioneval.CasePages, error) {
		cp := extractionevaldb.CasePageses
		entity, err := cp.Where(cp.CaseID.Eq(req.CaseID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, casePagesErrors.Map(err, req.CaseID.String())
		}

		return entity, nil
	})
}
