package equipmentmanufacturerrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var manufacturerErrors = dberror.Mapping{
	Entity: "EquipmentManufacturer",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_equipment_manu_name",
			Field:   "name",
			Message: "Equipment manufacturer with this name already exists in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) equipmentmanufacturerports.EquipmentManufacturerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-manufacturer-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *equipmentmanufacturerports.ListEquipmentManufacturersRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			em := equipmentmanufacturerdb.EquipmentManufacturers
			result, err := listquery.List(ctx, r.db.Keel(ctx), em.Query(),
				&listquery.Request[equipmentmanufacturer.EquipmentManufacturer]{
					Filter: req.Filter,
					Then:   []keel.Order{em.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list equipment manufacturers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *equipmentmanufacturerports.ListEquipmentManufacturerConnectionRequest,
) (*pagination.CursorListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			em := equipmentmanufacturerdb.EquipmentManufacturers
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), em.Query(),
				&listquery.Request[equipmentmanufacturer.EquipmentManufacturer]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.EquipmentManufacturerColumns,
				})
			if err != nil {
				r.l.Error("failed to list equipment manufacturers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentmanufacturer.EquipmentManufacturer, error) {
			err := equipmentmanufacturerdb.EquipmentManufacturers.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				r.l.Error("failed to create equipment manufacturer",
					zap.String("name", entity.Name), zap.Error(err))
				return nil, manufacturerErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentmanufacturer.EquipmentManufacturer, error) {
			err := equipmentmanufacturerdb.EquipmentManufacturers.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update equipment manufacturer",
					zap.String("id", entity.ID.String()), zap.Error(err))
				return nil, manufacturerErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req equipmentmanufacturerports.GetEquipmentManufacturerByIDRequest,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*equipmentmanufacturer.EquipmentManufacturer, error) {
			entity, err := equipmentmanufacturerdb.EquipmentManufacturers.Get(
				ctx,
				r.db.Keel(ctx),
				req.ID,
			)
			if err != nil {
				return nil, manufacturerErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req equipmentmanufacturerports.GetEquipmentManufacturersByIDsRequest,
) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
			if len(req.EquipmentManufacturerIDs) == 0 {
				return []*equipmentmanufacturer.EquipmentManufacturer{}, nil
			}

			em := equipmentmanufacturerdb.EquipmentManufacturers
			entities, err := em.Where(em.ID.In(req.EquipmentManufacturerIDs...)).
				Limit(len(req.EquipmentManufacturerIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get equipment manufacturers", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			em := equipmentmanufacturerdb.EquipmentManufacturers
			q := em.Pick(em.ID, em.CreatedAt, em.Name, em.Description, em.Status).
				Where(em.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(ctx, r.db.Keel(ctx), q, req, func(s string) keel.Pred {
				return keel.Or(em.Name.ContainsFold(s), em.Description.ContainsFold(s))
			})
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *equipmentmanufacturerports.BulkUpdateEquipmentManufacturerStatusRequest,
) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*equipmentmanufacturer.EquipmentManufacturer, error) {
			em := equipmentmanufacturerdb.EquipmentManufacturers
			entities, err := listquery.UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				em.Where(em.ID.In(req.EquipmentManufacturerIDs...)),
				em.Status.To(req.Status),
			)
			if err != nil {
				r.l.Error("failed to bulk update equipment manufacturer status", zap.Error(err))
				return nil, err
			}

			if err = dberror.CheckBulkAffected(
				int64(len(entities)),
				"EquipmentManufacturer",
				req.EquipmentManufacturerIDs,
			); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}
