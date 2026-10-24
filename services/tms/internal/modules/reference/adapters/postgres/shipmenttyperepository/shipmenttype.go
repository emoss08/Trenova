package shipmenttyperepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var shipmentTypeErrors = dberror.Mapping{
	Entity: "ShipmentType",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_shipment_types_code",
			Field:   "code",
			Message: "Shipment type with this code already exists in your organization",
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

func New(p Params) shipmenttypeports.ShipmentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-type-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *shipmenttypeports.ListShipmentTypesRequest,
) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
			st := shipmenttypedb.ShipmentTypes
			result, err := listquery.List(ctx, r.db.Keel(ctx), st.Query(),
				&listquery.Request[shipmenttype.ShipmentType]{
					Filter: req.Filter,
					Then:   []keel.Order{st.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list shipment types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *shipmenttypeports.ListShipmentTypesConnectionRequest,
) (*pagination.CursorListResult[*shipmenttype.ShipmentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*shipmenttype.ShipmentType], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				shipmenttypedb.ShipmentTypes.Query(),
				&listquery.Request[shipmenttype.ShipmentType]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.ShipmentTypeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list shipment types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmenttype.ShipmentType, error) {
		if err := shipmenttypedb.ShipmentTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create shipment type",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, shipmentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) (*shipmenttype.ShipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmenttype.ShipmentType, error) {
		if err := shipmenttypedb.ShipmentTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update shipment type",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, shipmentTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req shipmenttypeports.GetShipmentTypeByIDRequest,
) (*shipmenttype.ShipmentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmenttype.ShipmentType, error) {
		entity, err := shipmenttypedb.ShipmentTypes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, shipmentTypeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req shipmenttypeports.GetShipmentTypesByIDsRequest,
) ([]*shipmenttype.ShipmentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmenttype.ShipmentType, error) {
		if len(req.ShipmentTypeIDs) == 0 {
			return []*shipmenttype.ShipmentType{}, nil
		}

		st := shipmenttypedb.ShipmentTypes
		entities, err := st.Where(st.ID.In(req.ShipmentTypeIDs...)).
			Limit(len(req.ShipmentTypeIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get shipment types", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *shipmenttypeports.BulkUpdateShipmentTypeStatusRequest,
) ([]*shipmenttype.ShipmentType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipmenttype.ShipmentType, error) {
		st := shipmenttypedb.ShipmentTypes
		entities, err := listquery.UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			st.Where(st.ID.In(req.ShipmentTypeIDs...)),
			st.Status.To(req.Status),
		)
		if err != nil {
			r.l.Error("failed to bulk update shipment type status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "ShipmentType", req.ShipmentTypeIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *shipmenttypeports.ShipmentTypeSelectOptionsRequest,
) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
			st := shipmenttypedb.ShipmentTypes
			q := st.Pick(st.ID, st.Status, st.Code, st.Description, st.Color).
				Where(st.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(st.Code.ContainsFold(s), st.Description.ContainsFold(s))
				},
			)
		},
	)
}
