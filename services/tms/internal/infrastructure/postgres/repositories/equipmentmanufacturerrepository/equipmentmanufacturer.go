package equipmentmanufacturerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
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
	cols := buncolgen.EquipmentManufacturerColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.EquipmentManufacturerTable.Alias,
		req.Filter,
		(*equipmentmanufacturer.EquipmentManufacturer)(nil),
	)

	q = q.Apply(buncolgen.EquipmentManufacturerApplyTenant(req.Filter.TenantInfo)).
		Order(cols.CreatedAt.OrderDesc())

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEquipmentManufacturersRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make(
		[]*equipmentmanufacturer.EquipmentManufacturer,
		0,
		req.Filter.Pagination.SafeLimit(),
	)
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
	cols := buncolgen.EquipmentManufacturerColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where(cols.Version.Eq(), ov).
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
	cols := buncolgen.EquipmentManufacturerColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EquipmentManufacturerScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
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
			return buncolgen.EquipmentManufacturerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.EquipmentManufacturerColumns.ID.In(), bun.List(req.EquipmentManufacturerIDs))
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
	cols := buncolgen.EquipmentManufacturerColumns

	return dbhelper.SelectOptions[*equipmentmanufacturer.EquipmentManufacturer](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Name,
				cols.Description,
				cols.Status,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
			},
			EntityName:       "EquipmentManufacturer",
			SearchColumnRefs: []buncolgen.Column{cols.Name, cols.Description},
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
			return buncolgen.EquipmentManufacturerScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.EquipmentManufacturerColumns.ID.In(), bun.List(req.EquipmentManufacturerIDs))
		}).
		Set(buncolgen.EquipmentManufacturerColumns.Status.Set(), req.Status).
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
