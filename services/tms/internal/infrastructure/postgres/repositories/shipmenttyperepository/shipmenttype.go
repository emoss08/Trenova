package shipmenttyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
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

func New(p Params) repositories.ShipmentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipmenttype-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentTypesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"sht",
		req.Filter,
		(*shipmenttype.ShipmentType)(nil),
	)

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListShipmentTypesRequest,
) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*shipmenttype.ShipmentType, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count shipment types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*shipmenttype.ShipmentType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create shipment type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("description", entity.Description),
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
		log.Error("failed to update shipment type", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "ShipmentType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetShipmentTypeByIDRequest,
) (*shipmenttype.ShipmentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(shipmenttype.ShipmentType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sht.id = ?", req.ID).
				Where("sht.organization_id = ?", req.TenantInfo.OrgID).
				Where("sht.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get shipment type", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ShipmentType")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.ShipmentTypeSelectOptionsRequest,
) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
	return dbhelper.SelectOptions[*shipmenttype.ShipmentType](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"description",
				"color",
			},
			OrgColumn: "sht.organization_id",
			BuColumn:  "sht.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("sht.status = ?", domaintypes.StatusActive)
			},
			EntityName: "ShipmentType",
			SearchColumns: []string{
				"sht.code",
				"sht.description",
			},
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateShipmentTypeStatusRequest,
) ([]*shipmenttype.ShipmentType, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*shipmenttype.ShipmentType, 0, len(req.ShipmentTypeIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("sht.organization_id = ?", req.TenantInfo.OrgID).
				Where("sht.business_unit_id = ?", req.TenantInfo.BuID).
				Where("sht.id IN (?)", bun.In(req.ShipmentTypeIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update shipment type status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "ShipmentType", req.ShipmentTypeIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetShipmentTypesByIDsRequest,
) ([]*shipmenttype.ShipmentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*shipmenttype.ShipmentType, 0, len(req.ShipmentTypeIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sht.organization_id = ?", req.TenantInfo.OrgID).
				Where("sht.business_unit_id = ?", req.TenantInfo.BuID).
				Where("sht.id IN (?)", bun.In(req.ShipmentTypeIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get shipment types", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ShipmentType")
	}

	return entities, nil
}
