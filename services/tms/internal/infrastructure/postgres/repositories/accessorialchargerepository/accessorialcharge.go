package accessorialchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.AccessorialChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accessorialcharge-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAccessorialChargeRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"acc",
		req.Filter,
		(*accessorialcharge.AccessorialCharge)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAccessorialChargeRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*accessorialcharge.AccessorialCharge, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count accessorial charges", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create accessorial charge", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update accessorial charge", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AccessorialCharge", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(accessorialcharge.AccessorialCharge)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("acc.id = ?", req.ID).
				Where("acc.organization_id = ?", req.TenantInfo.OrgID).
				Where("acc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get accessorial charge", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AccessorialCharge")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return dbhelper.SelectOptions[*accessorialcharge.AccessorialCharge](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"description",
				"status",
				"method",
				"rate_unit",
				"amount",
			},
			OrgColumn:     "acc.organization_id",
			BuColumn:      "acc.business_unit_id",
			SearchColumns: []string{"acc.code", "acc.description"},
			EntityName:    "AccessorialCharge",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("acc.status = ?", domaintypes.StatusActive)
			},
		},
	)
}
