package fleetcoderepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
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

func New(p Params) repositories.FleetCodeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fleet-code-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFleetCodesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fc",
		req.Filter,
		(*fleetcode.FleetCode)(nil),
	)

	if req.IncludeManagerDetails {
		q = q.Relation("Manager")
	}

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFleetCodesRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*fleetcode.FleetCode, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count fleet codes", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*fleetcode.FleetCode]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create fleet code", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
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
		log.Error("failed to update fleet code", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FleetCode", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetFleetCodeByIDRequest,
) (*fleetcode.FleetCode, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fleetcode.FleetCode)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fc.id = ?", req.ID).
				Where("fc.organization_id = ?", req.TenantInfo.OrgID).
				Where("fc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get fleet code", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FleetCode")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	return dbhelper.SelectOptions[*fleetcode.FleetCode](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns:       []string{"id", "code", "description", "status", "manager_id"},
			OrgColumn:     "fc.organization_id",
			BuColumn:      "fc.business_unit_id",
			SearchColumns: []string{"fc.code", "fc.description"},
			EntityName:    "FleetCode",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("fc.status = ?", domaintypes.StatusActive).Relation("Manager")
			},
		},
	)
}
