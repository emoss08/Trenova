package equipmenttyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
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

func New(p Params) repositories.EquipmentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-type-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEquipmentTypesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"et",
		req.Filter,
		(*equipmenttype.EquipmentType)(nil),
	)

	if len(req.Classes) > 0 {
		var validClasses []equipmenttype.Class
		for _, class := range req.Classes {
			if class != "" {
				validClasses = append(validClasses, equipmenttype.Class(class))
			}
		}

		if len(validClasses) > 0 {
			q = q.Where("et.class IN (?)", bun.In(validClasses))
		}
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEquipmentTypesRequest,
) (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*equipmenttype.EquipmentType, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count equipment types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*equipmenttype.EquipmentType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create equipment type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
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
		log.Error("failed to update equipment type", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "EquipmentType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetEquipmentTypeByIDRequest,
) (*equipmenttype.EquipmentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(equipmenttype.EquipmentType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("et.id = ?", req.ID).
				Where("et.organization_id = ?", req.TenantInfo.OrgID).
				Where("et.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get equipment type", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EquipmentType")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.EquipmentTypeSelectOptionsRequest,
) (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
	return dbhelper.SelectOptions[*equipmenttype.EquipmentType](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"description",
				"class",
				"color",
				"status",
			},
			OrgColumn: "et.organization_id",
			BuColumn:  "et.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if len(req.Classes) > 0 {
					var validClasses []equipmenttype.Class
					for _, class := range req.Classes {
						if class != "" {
							validClasses = append(validClasses, equipmenttype.Class(class))
						}
					}

					if len(validClasses) > 0 {
						q = q.Where("et.class IN (?)", bun.In(validClasses))
					}
				}

				return q.Where("et.status = ?", domaintypes.StatusActive)
			},
			EntityName: "EquipmentType",
			SearchColumns: []string{
				"et.code",
				"et.description",
			},
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateEquipmentTypeStatusRequest,
) ([]*equipmenttype.EquipmentType, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*equipmenttype.EquipmentType, 0, len(req.EquipmentTypeIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("et.organization_id = ?", req.TenantInfo.OrgID).
				Where("et.business_unit_id = ?", req.TenantInfo.BuID).
				Where("et.id IN (?)", bun.In(req.EquipmentTypeIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update equipment type status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "EquipmentType", req.EquipmentTypeIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetEquipmentTypesByIDsRequest,
) ([]*equipmenttype.EquipmentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*equipmenttype.EquipmentType, 0, len(req.EquipmentTypeIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("et.organization_id = ?", req.TenantInfo.OrgID).
				Where("et.business_unit_id = ?", req.TenantInfo.BuID).
				Where("et.id IN (?)", bun.In(req.EquipmentTypeIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get equipment types", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EquipmentType")
	}

	return entities, nil
}
