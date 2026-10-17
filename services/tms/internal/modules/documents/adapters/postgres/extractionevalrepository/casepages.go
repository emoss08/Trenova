package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/extractionevalcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const casePagesEntity = "ExtractionEvalCasePages"

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
		cols := extractionevalcol.CasePagesColumns
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On(`CONFLICT ("organization_id", "business_unit_id", "case_id", "text_source") DO UPDATE`).
			Set(cols.Pages.SetExcluded()).
			Set(cols.PageCount.SetExcluded()).
			Set(cols.InputHash.SetExcluded()).
			Set(cols.ConverterVersion.SetExcluded()).
			Set(cols.FrozenByID.SetExcluded()).
			Set(cols.FrozenAt.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to save extraction eval case pages", zap.Error(err))

			return nil, fmt.Errorf("save extraction eval case pages: %w", err)
		}

		return entity, nil
	})
}

func (r *casePagesRepository) Get(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalCasePagesRequest,
) (*extractioneval.CasePages, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractioneval.CasePages, error) {
		entity := new(extractioneval.CasePages)
		cols := extractionevalcol.CasePagesColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return extractionevalcol.CasePagesScopeTenant(sq, req.TenantInfo).
					Where(cols.CaseID.Eq(), req.CaseID).
					Where(cols.TextSource.Eq(), req.TextSource)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, casePagesEntity)
		}

		return entity, nil
	})
}
