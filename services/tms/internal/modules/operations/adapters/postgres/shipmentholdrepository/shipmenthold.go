package shipmentholdrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHoldLimit       = 50
	activeHoldByTypeIndex  = "ux_shipment_holds_active_by_type"
	shipmentHoldEntityName = "Shipment hold"
)

var holdErrors = dberror.Mapping{Entity: shipmentHoldEntityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentholdports.ShipmentHoldRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-hold-repository"),
	}
}

func holdQuery() *keel.Query[*shipmenthold.ShipmentHold] {
	h := shipmentholddb.ShipmentHolds
	return h.Load(h.HoldReason(), h.CreatedBy(), h.ReleasedBy())
}

func (r *repository) ListByShipmentID(
	ctx context.Context,
	req *shipmentholdports.ListShipmentHoldsRequest,
) (*pagination.ListResult[*shipmenthold.ShipmentHold], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipmenthold.ShipmentHold], error) {
			if req.Filter.Pagination.Limit <= 0 {
				req.Filter.Pagination.Limit = defaultHoldLimit
			}

			h := shipmentholddb.ShipmentHolds
			counted := h.Where(h.ShipmentID.Eq(req.ShipmentID))
			rows := holdQuery().
				Where(h.ShipmentID.Eq(req.ShipmentID)).
				Order(
					keel.Desc(h.ReleasedAt.IsNull()),
					h.StartedAt.Desc(),
					h.ReleasedAt.Coalesce(0).Desc(),
				).
				Limit(req.Filter.Pagination.Limit).
				Offset(req.Filter.Pagination.Offset)

			var holds []*shipmenthold.ShipmentHold
			var total keel.RowCount
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &holds),
				keel.CountEstimateInto(counted, pagination.ExactTotalUpTo, &total),
			); err != nil {
				return nil, err
			}

			return listquery.Result(holds, total), nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *shipmentholdports.GetShipmentHoldByIDRequest,
) (*shipmenthold.ShipmentHold, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmenthold.ShipmentHold, error) {
		h := shipmentholddb.ShipmentHolds
		entity, err := holdQuery().
			Where(h.ID.Eq(req.HoldID), h.ShipmentID.Eq(req.ShipmentID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, holdErrors.Map(err, req.HoldID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipmenthold.ShipmentHold,
) (*shipmenthold.ShipmentHold, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmenthold.ShipmentHold, error) {
		if err := shipmentholddb.ShipmentHolds.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) &&
				dberror.ExtractConstraintName(err) == activeHoldByTypeIndex {
				return nil, errortypes.NewBusinessError("Shipment already has an active hold of this type").
					WithParam("shipmentId", entity.ShipmentID.String())
			}
			return nil, fmt.Errorf(
				"insert shipment hold: %w",
				holdErrors.Map(err, entity.ID.String()),
			)
		}

		return r.GetByID(ctx, &shipmentholdports.GetShipmentHoldByIDRequest{
			HoldID:     entity.ID,
			ShipmentID: entity.ShipmentID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipmenthold.ShipmentHold,
) (*shipmenthold.ShipmentHold, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmenthold.ShipmentHold, error) {
		h := shipmentholddb.ShipmentHolds
		n, err := h.Where(
			h.ID.Eq(entity.ID),
			h.ShipmentID.Eq(entity.ShipmentID),
			h.ReleasedAt.IsNull(),
			h.Version.Eq(entity.Version),
		).Update(
			ctx,
			r.db.Keel(ctx),
			h.StartedAt.To(entity.StartedAt),
			h.Severity.To(entity.Severity),
			h.Notes.To(entity.Notes),
			h.BlocksDispatch.To(entity.BlocksDispatch),
			h.BlocksDelivery.To(entity.BlocksDelivery),
			h.BlocksBilling.To(entity.BlocksBilling),
			h.VisibleToCustomer.To(entity.VisibleToCustomer),
		)
		if err != nil {
			return nil, fmt.Errorf("update shipment hold: %w", err)
		}
		if err = dberror.CheckAffected(n, shipmentHoldEntityName, entity.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentholdports.GetShipmentHoldByIDRequest{
			HoldID:     entity.ID,
			ShipmentID: entity.ShipmentID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Release(
	ctx context.Context,
	entity *shipmenthold.ShipmentHold,
) (*shipmenthold.ShipmentHold, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmenthold.ShipmentHold, error) {
		h := shipmentholddb.ShipmentHolds
		n, err := h.Where(
			h.ID.Eq(entity.ID),
			h.ShipmentID.Eq(entity.ShipmentID),
			h.ReleasedAt.IsNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			h.ReleasedAt.ToExpr(keel.Val(entity.ReleasedAt)),
			h.ReleasedByID.ToExpr(keel.Val(entity.ReleasedByID)),
		)
		if err != nil {
			return nil, fmt.Errorf("release shipment hold: %w", err)
		}
		if err = dberror.CheckAffected(n, shipmentHoldEntityName, entity.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, &shipmentholdports.GetShipmentHoldByIDRequest{
			HoldID:     entity.ID,
			ShipmentID: entity.ShipmentID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) HasActiveDispatchHold(
	ctx context.Context,
	req *shipmentholdports.ActiveShipmentHoldRequest,
) (bool, error) {
	return r.hasActiveBlockingHold(ctx, req, shipmentholddb.ShipmentHolds.BlocksDispatch)
}

func (r *repository) HasActiveDeliveryHold(
	ctx context.Context,
	req *shipmentholdports.ActiveShipmentHoldRequest,
) (bool, error) {
	return r.hasActiveBlockingHold(ctx, req, shipmentholddb.ShipmentHolds.BlocksDelivery)
}

func (r *repository) hasActiveBlockingHold(
	ctx context.Context,
	req *shipmentholdports.ActiveShipmentHoldRequest,
	blocks keel.BoolCol,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		h := shipmentholddb.ShipmentHolds
		exists, err := h.Where(
			h.ShipmentID.Eq(req.ShipmentID),
			h.ReleasedAt.IsNull(),
			h.Severity.Eq(holdreason.HoldSeverityBlocking),
			blocks.IsTrue(),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check active shipment hold: %w", err)
		}

		return exists, nil
	})
}

func tenantInfo(entity *shipmenthold.ShipmentHold) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
