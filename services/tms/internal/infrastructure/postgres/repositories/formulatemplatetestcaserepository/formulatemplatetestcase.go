package formulatemplatetestcaserepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.FormulaTemplateTestCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-test-case-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.TestCase,
) (*formulatemplate.TestCase, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	_, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create formula template test case", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulatemplate.TestCase,
) (*formulatemplate.TestCase, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Value("description", "?", entity.Description).
		Value("variables", "?", entity.Variables).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update formula template test case", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FormulaTemplateTestCase", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.GetTestCaseByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*formulatemplate.TestCase)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("ftc.id = ?", req.TestCaseID).
				Where("ftc.template_id = ?", req.TemplateID).
				Where("ftc.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete formula template test case", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "FormulaTemplateTestCase", req.TestCaseID.String())
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTestCaseByIDRequest,
) (*formulatemplate.TestCase, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	entity := new(formulatemplate.TestCase)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ftc.id = ?", req.TestCaseID).
				Where("ftc.template_id = ?", req.TemplateID).
				Where("ftc.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get formula template test case", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FormulaTemplateTestCase")
	}

	return entity, nil
}

func (r *repository) ListByTemplate(
	ctx context.Context,
	req repositories.ListTestCasesRequest,
) ([]*formulatemplate.TestCase, error) {
	log := r.l.With(
		zap.String("operation", "ListByTemplate"),
		zap.String("templateID", req.TemplateID.String()),
	)

	entities := make([]*formulatemplate.TestCase, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ftc.template_id = ?", req.TemplateID).
				Where("ftc.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Order("ftc.created_at ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to list formula template test cases", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
