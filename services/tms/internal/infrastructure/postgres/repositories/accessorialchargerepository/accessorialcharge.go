package accessorialchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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
	cols := buncolgen.AccessorialChargeColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.AccessorialChargeTable.Alias,
		req.Filter,
		(*accessorialcharge.AccessorialCharge)(nil),
	)

	return q.Apply(buncolgen.AccessorialChargeApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
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
	cols := buncolgen.AccessorialChargeColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where(cols.Version.Eq(), ov).
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
	cols := buncolgen.AccessorialChargeColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AccessorialChargeScopeTenant(sq, *req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
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
	cols := buncolgen.AccessorialChargeColumns

	return dbhelper.SelectOptions[*accessorialcharge.AccessorialCharge](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Description,
				cols.Status,
				cols.Method,
				cols.RateUnit,
				cols.Amount,
			},
			OrgColumnRef:     &cols.OrganizationID,
			BuColumnRef:      &cols.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Description},
			EntityName:       "AccessorialCharge",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
			},
		},
	)
}
