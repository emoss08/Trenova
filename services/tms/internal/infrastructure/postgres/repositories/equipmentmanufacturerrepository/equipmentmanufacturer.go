package equipmentmanufacturerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
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

func New(p Params) repositories.EquipmentManufacturerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-manufacturer-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEquipmentManufacturersRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"em",
		req.Filter,
		(*equipmentmanufacturer.EquipmentManufacturer)(nil),
	)

	q = q.Order("em.created_at DESC")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEquipmentManufacturersRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*equipmentmanufacturer.EquipmentManufacturer, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count equipment manufacturers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create equipment manufacturer", zap.Error(err))
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
		log.Error("failed to update equipment manufacturer", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "EquipmentManufacturer", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetEquipmentManufacturerByIDRequest,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("em.id = ?", req.ID).
				Where("em.organization_id = ?", req.TenantInfo.OrgID).
				Where("em.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get equipment manufacturer", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EquipmentManufacturer")
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetEquipmentManufacturersByIDsRequest,
) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make(
		[]*equipmentmanufacturer.EquipmentManufacturer,
		0,
		len(req.EquipmentManufacturerIDs),
	)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("em.organization_id = ?", req.TenantInfo.OrgID).
				Where("em.business_unit_id = ?", req.TenantInfo.BuID).
				Where("em.id IN (?)", bun.In(req.EquipmentManufacturerIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get equipment manufacturers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EquipmentManufacturer")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	return dbhelper.SelectOptions[*equipmentmanufacturer.EquipmentManufacturer](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns:   []string{"id", "name", "description", "status"},
			OrgColumn: "em.organization_id",
			BuColumn:  "em.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("em.status = ?", domaintypes.StatusActive)
			},
			EntityName:    "EquipmentManufacturer",
			SearchColumns: []string{"em.name", "em.description"},
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateEquipmentManufacturerStatusRequest,
) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make(
		[]*equipmentmanufacturer.EquipmentManufacturer,
		0,
		len(req.EquipmentManufacturerIDs),
	)
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("em.organization_id = ?", req.TenantInfo.OrgID).
				Where("em.business_unit_id = ?", req.TenantInfo.BuID).
				Where("em.id IN (?)", bun.In(req.EquipmentManufacturerIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update equipment manufacturer status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "EquipmentManufacturer", req.EquipmentManufacturerIDs); err != nil {
		return nil, err
	}

	return entities, nil
}
