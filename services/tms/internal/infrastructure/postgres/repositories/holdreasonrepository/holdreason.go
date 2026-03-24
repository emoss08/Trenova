package holdreasonrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.HoldReasonRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hold-reason-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHoldReasonRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"hr",
		req.Filter,
		(*holdreason.HoldReason)(nil),
	)

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListHoldReasonRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*holdreason.HoldReason, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count hold reasons", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*holdreason.HoldReason]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetHoldReasonByIDRequest,
) (*holdreason.HoldReason, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(holdreason.HoldReason)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("hr.id = ?", req.ID).
				Where("hr.organization_id = ?", req.TenantInfo.OrgID).
				Where("hr.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get hold reason", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "HoldReason")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *holdreason.HoldReason,
) (*holdreason.HoldReason, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create hold reason", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *holdreason.HoldReason,
) (*holdreason.HoldReason, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update hold reason", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "HoldReason", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.HoldReasonSelectOptionsRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return dbhelper.SelectOptions[*holdreason.HoldReason](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"label",
				"type",
			},
			OrgColumn: "hr.organization_id",
			BuColumn:  "hr.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("hr.active = ?", true)
			},
			EntityName: "HoldReason",
			SearchColumns: []string{
				"hr.code",
				"hr.label",
			},
		},
	)
}
