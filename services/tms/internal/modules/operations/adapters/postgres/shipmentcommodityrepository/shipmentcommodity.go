package shipmentcommodityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var commodityErrors = dberror.Mapping{
	Entity: "Shipment commodity",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_shipment_commodities_commodity",
			Field:   "commodityId",
			Message: "Commodity does not exist in your organization",
		},
		{
			Name:    "fk_shipment_commodities_shipment",
			Field:   "shipmentId",
			Message: "Shipment does not exist in your organization",
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

func New(p Params) shipmentports.ShipmentCommodityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-commodity-repository"),
	}
}

func (r *repository) SyncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.syncForShipment(ctx, entity)
	})
}

func (r *repository) syncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	existing, err := r.getExistingCommodities(ctx, entity)
	if err != nil {
		return err
	}

	kept := make(map[pulid.ID]struct{}, len(entity.Commodities))
	inserts := make([]*shipment.ShipmentCommodity, 0, len(entity.Commodities))
	for _, commodity := range entity.Commodities {
		if commodity == nil {
			continue
		}

		normalizeCommodity(entity, commodity)

		switch {
		case commodity.ID.IsNil():
			commodity.ID = pulid.MustNew("sc_")
			inserts = append(inserts, commodity)
		case existing[commodity.ID] != nil:
			if err = r.updateCommodity(ctx, commodity, existing[commodity.ID]); err != nil {
				return err
			}
			kept[commodity.ID] = struct{}{}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown commodity").
				WithParam("shipmentCommodityId", commodity.ID.String())
		}
	}

	if len(inserts) > 0 {
		if err = shipmentdb.ShipmentCommodities.CreateMany(ctx, r.db.Keel(ctx), inserts); err != nil {
			return fmt.Errorf(
				"insert shipment commodities: %w",
				commodityErrors.Map(err, inserts[0].ID.String()),
			)
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existing))
	for id := range existing {
		if _, ok := kept[id]; ok {
			continue
		}
		if inPayload(entity, id) {
			continue
		}
		deleteIDs = append(deleteIDs, id)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	sc := shipmentdb.ShipmentCommodities
	if _, err = sc.Where(sc.ID.In(deleteIDs...), sc.ShipmentID.Eq(entity.ID)).
		Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete shipment commodities: %w", err)
	}

	return nil
}

func (r *repository) getExistingCommodities(
	ctx context.Context,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.ShipmentCommodity, error) {
	sc := shipmentdb.ShipmentCommodities
	commodities, err := sc.Where(sc.ShipmentID.Eq(entity.ID)).All(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("get existing shipment commodities: %w", err)
	}

	result := make(map[pulid.ID]*shipment.ShipmentCommodity, len(commodities))
	for _, commodity := range commodities {
		result[commodity.ID] = commodity
	}

	return result, nil
}

func normalizeCommodity(entity *shipment.Shipment, commodity *shipment.ShipmentCommodity) {
	commodity.ShipmentID = entity.ID
	commodity.OrganizationID = entity.OrganizationID
	commodity.BusinessUnitID = entity.BusinessUnitID
}

func (r *repository) updateCommodity(
	ctx context.Context,
	commodity *shipment.ShipmentCommodity,
	existing *shipment.ShipmentCommodity,
) error {
	sc := shipmentdb.ShipmentCommodities
	n, err := sc.Where(
		sc.ID.Eq(commodity.ID),
		sc.ShipmentID.Eq(commodity.ShipmentID),
		sc.Version.Eq(existing.Version),
	).Update(
		ctx,
		r.db.Keel(ctx),
		sc.CommodityID.To(commodity.CommodityID),
		sc.Weight.To(commodity.Weight),
		sc.Pieces.To(commodity.Pieces),
		sc.LengthFeet.ToExpr(keel.Val(commodity.LengthFeet)),
		sc.WidthFeet.ToExpr(keel.Val(commodity.WidthFeet)),
		sc.HeightFeet.ToExpr(keel.Val(commodity.HeightFeet)),
	)
	if err != nil {
		return fmt.Errorf(
			"update shipment commodity %s: %w",
			commodity.ID,
			commodityErrors.Map(err, commodity.ID.String()),
		)
	}

	if err = dberror.CheckAffected(n, "Shipment commodity", commodity.ID.String()); err != nil {
		return err
	}

	commodity.Version = existing.Version + 1
	commodity.CreatedAt = existing.CreatedAt
	commodity.UpdatedAt = timeutils.NowUnix()

	return nil
}

func inPayload(entity *shipment.Shipment, id pulid.ID) bool {
	for _, commodity := range entity.Commodities {
		if commodity != nil && commodity.ID == id {
			return true
		}
	}

	return false
}
