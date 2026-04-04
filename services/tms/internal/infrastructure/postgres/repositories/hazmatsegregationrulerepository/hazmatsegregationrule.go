package hazmatsegregationrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.HazmatSegregationRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazmat-segregation-rule-repository"),
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

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListHazmatSegregationRuleRequest,
) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*hazmatsegregationrule.HazmatSegregationRule, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count hazmat segregation rules", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetHazmatSegregationRuleByIDRequest,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
	err := r.db.DB().NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("hsr.id = ?", req.ID).
				Where("hsr.organization_id = ?", req.TenantInfo.OrgID).
				Where("hsr.business_unit_id = ?", req.TenantInfo.BuID)
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to get hazmat segregation rule", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "HazmatSegregationRule")
	}

	return entity, nil
}

func (r *repository) ListActiveByTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*hazmatsegregationrule.HazmatSegregationRule, error) {
	entities := make([]*hazmatsegregationrule.HazmatSegregationRule, 0)

	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Where("hsr.organization_id = ?", tenantInfo.OrgID).
		Where("hsr.business_unit_id = ?", tenantInfo.BuID).
		Where("hsr.status = ?", domaintypes.StatusActive).
		Order("hsr.name ASC", "hsr.id ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create hazmat segregation rule", zap.Error(err))
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
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update hazmat segregation rule", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "HazmatSegregationRule", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
