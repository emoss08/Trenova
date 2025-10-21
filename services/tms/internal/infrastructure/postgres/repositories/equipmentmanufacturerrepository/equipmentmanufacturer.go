package equipmentmanufacturerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
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

func NewRepository(p Params) repositories.EquipmentManufacturerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipmentmanufacturer-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEquipmentManufacturerRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"em",
		req.Filter,
		(*equipmentmanufacturer.EquipmentManufacturer)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEquipmentManufacturerRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
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

	entities := make([]*equipmentmanufacturer.EquipmentManufacturer, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan equipment manufacturers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetEquipmentManufacturerByIDRequest,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("em.id = ?", req.ID).
				Where("em.organization_id = ?", req.OrgID).
				Where("em.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Equipment Manufacturer")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
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
		log.Error("failed to insert equipment manufacturer", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
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
		Where("em.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update equipment manufacturer", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Equipment Manufacturer", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
