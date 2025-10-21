package tractorrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
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

func NewRepository(p Params) repositories.TractorRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tractor-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	req repositories.TractorFilterOptions,
) *bun.SelectQuery {
	if req.IncludeWorkerDetails {
		q = q.RelationWithOpts("PrimaryWorker", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("WorkerProfile")
			},
		})

		q = q.RelationWithOpts("SecondaryWorker", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("WorkerProfile")
			},
		})
	}

	if req.IncludeEquipmentDetails {
		q = q.Relation("FleetCode")
	}

	if req.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if req.Status != "" {
		status, err := domain.EquipmentStatusFromString(req.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", req.Status))
			return q
		}

		q = q.Where("tr.status = ?", status)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTractorRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"tr",
		req.Filter,
		(*tractor.Tractor)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
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

	entities := make([]*tractor.Tractor, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan tractors", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tractor.Tractor]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.TractorID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(tractor.Tractor)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tr.id = ?", req.TractorID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		}).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Tractor")
	}

	return entity, nil
}

func (r *repository) GetByPrimaryWorkerID(
	ctx context.Context,
	req repositories.GetTractorByPrimaryWorkerIDRequest,
) (*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "GetByPrimaryWorkerID"),
		zap.Any("request", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(tractor.Tractor)

	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tr.primary_worker_id = ?", req.WorkerID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Tractor")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
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
		log.Error("failed to insert tractor", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
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
		Where("tr.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update tractor", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Tractor", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Assignment(
	ctx context.Context,
	req repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	log := r.l.With(
		zap.String("operation", "Assignment"),
		zap.String("tractorID", req.TractorID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(tractor.Tractor)

	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tr.id = ?", req.TractorID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Tractor")
	}

	return &repositories.AssignmentResponse{
		PrimaryWorkerID:   entity.PrimaryWorkerID,
		SecondaryWorkerID: entity.SecondaryWorkerID,
	}, nil
}
