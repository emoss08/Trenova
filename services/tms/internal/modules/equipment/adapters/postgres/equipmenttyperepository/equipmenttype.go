package equipmenttyperepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var equipmentTypeErrors = dberror.Mapping{
	Entity: "EquipmentType",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_equipment_types_code",
			Field:   "code",
			Message: "Equipment type with this code already exists in your organization",
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

func New(p Params) equipmenttypeports.EquipmentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.equipment-type-repository"),
	}
}

func classFilter(classes []string) keel.Pred {
	valid := make([]equipmenttype.Class, 0, len(classes))
	for _, class := range classes {
		if class != "" {
			valid = append(valid, equipmenttype.Class(class))
		}
	}
	if len(valid) == 0 {
		return nil
	}

	return equipmenttypedb.EquipmentTypes.Class.In(valid...)
}

func (r *repository) List(
	ctx context.Context,
	req *equipmenttypeports.ListEquipmentTypesRequest,
) (*pagination.CursorListResult[*equipmenttype.EquipmentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*equipmenttype.EquipmentType], error) {
			et := equipmenttypedb.EquipmentTypes
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				et.Where(classFilter(req.Classes)),
				&listquery.Request[equipmenttype.EquipmentType]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.EquipmentTypeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list equipment types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*equipmenttype.EquipmentType, error) {
		if err := equipmenttypedb.EquipmentTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create equipment type",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, equipmentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*equipmenttype.EquipmentType, error) {
		if err := equipmenttypedb.EquipmentTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update equipment type",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, equipmentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req equipmenttypeports.GetEquipmentTypeByIDRequest,
) (*equipmenttype.EquipmentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*equipmenttype.EquipmentType, error) {
		entity, err := equipmenttypedb.EquipmentTypes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, equipmentTypeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *equipmenttypeports.EquipmentTypeSelectOptionsRequest,
) (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
			et := equipmenttypedb.EquipmentTypes
			q := et.Pick(et.ID, et.CreatedAt, et.Code, et.Description, et.Class, et.Color, et.Status).
				Where(et.Status.Eq(domaintypes.StatusActive), classFilter(req.Classes))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(et.Code.ContainsFold(s), et.Description.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *equipmenttypeports.BulkUpdateEquipmentTypeStatusRequest,
) ([]*equipmenttype.EquipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*equipmenttype.EquipmentType, error) {
		et := equipmenttypedb.EquipmentTypes
		entities, err := et.Where(et.ID.In(req.EquipmentTypeIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), et.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update equipment type status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkAffected(
			int64(len(entities)),
			"EquipmentType",
			req.EquipmentTypeIDs,
		); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req equipmenttypeports.GetEquipmentTypesByIDsRequest,
) ([]*equipmenttype.EquipmentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*equipmenttype.EquipmentType, error) {
		if len(req.EquipmentTypeIDs) == 0 {
			return []*equipmenttype.EquipmentType{}, nil
		}

		et := equipmenttypedb.EquipmentTypes
		entities, err := et.Where(et.ID.In(req.EquipmentTypeIDs...)).
			Limit(len(req.EquipmentTypeIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get equipment types", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
