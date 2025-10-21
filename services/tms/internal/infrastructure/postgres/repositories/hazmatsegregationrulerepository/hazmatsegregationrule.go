package hazmatsegregationrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(
	p Params,
) repositories.HazmatSegregationRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazmatsegregationrule-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHazmatSegregationRuleRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"hsr",
		req.Filter,
		(*hazmatsegregationrule.HazmatSegregationRule)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListHazmatSegregationRuleRequest,
) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*hazmatsegregationrule.HazmatSegregationRule, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan hazmat segregation rules", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetHazmatSegregationRuleByIDRequest,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("hsr.id = ?", req.ID).
				Where("hsr.organization_id = ?", req.OrgID).
				Where("hsr.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Hazmat Segregation Rule")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert hazmat segregation rule", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("hsr.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update hazmat segregation rule", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Hazmat Segregation Rule", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
