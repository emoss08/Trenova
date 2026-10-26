package hazmatsegregationrulerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule/hazmatsegregationruledb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var hazmatSegregationRuleErrors = dberror.Mapping{Entity: "HazmatSegregationRule"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) hazmatsegregationruleports.HazmatSegregationRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazmat-segregation-rule-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *hazmatsegregationruleports.ListHazmatSegregationRuleRequest,
) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
			hsr := hazmatsegregationruledb.HazmatSegregationRules
			result, err := listquery.List(ctx, r.db.Keel(ctx), hsr.Query(),
				&listquery.Request[hazmatsegregationrule.HazmatSegregationRule]{
					Filter: req.Filter,
					Then:   []keel.Order{hsr.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list hazmat segregation rules", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req hazmatsegregationruleports.GetHazmatSegregationRuleByIDRequest,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*hazmatsegregationrule.HazmatSegregationRule, error) {
			hsr := hazmatsegregationruledb.HazmatSegregationRules
			entity, err := hsr.Where(hsr.ID.Eq(req.ID)).
				Cache(resultcache.Reference()...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, hazmatSegregationRuleErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListActiveByTenant(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*hazmatsegregationrule.HazmatSegregationRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*hazmatsegregationrule.HazmatSegregationRule, error) {
			entities, err := activeRules().
				Cache(resultcache.Reference()...).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active hazmat segregation rules", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func activeRules() *keel.Query[*hazmatsegregationrule.HazmatSegregationRule] {
	hsr := hazmatsegregationruledb.HazmatSegregationRules

	return hsr.Where(hsr.Status.Eq(domaintypes.StatusActive)).Order(hsr.Name.Asc(), hsr.ID.Asc())
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *hazmatsegregationruleports.ListHazmatSegregationRuleConnectionRequest,
) (*pagination.CursorListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				hazmatsegregationruledb.HazmatSegregationRules.Query(),
				&listquery.Request[hazmatsegregationrule.HazmatSegregationRule]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.HazmatSegregationRuleColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list hazmat segregation rules", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*hazmatsegregationrule.HazmatSegregationRule, error) {
			err := hazmatsegregationruledb.HazmatSegregationRules.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				r.l.Error(
					"failed to create hazmat segregation rule",
					zap.String("name", entity.Name),
					zap.Error(err),
				)
				return nil, hazmatSegregationRuleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*hazmatsegregationrule.HazmatSegregationRule, error) {
			err := hazmatsegregationruledb.HazmatSegregationRules.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update hazmat segregation rule",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, hazmatSegregationRuleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
