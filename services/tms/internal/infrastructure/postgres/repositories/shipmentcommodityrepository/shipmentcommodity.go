package shipmentcommodityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.ShipmentCommodityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-commodity-repository"),
	}
}

func (r *repository) SyncForShipment(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	existingCommodities, err := r.getExistingCommodities(ctx, tx, entity)
	if err != nil {
		return err
	}

	updatedCommodityIDs := make(map[pulid.ID]struct{}, len(entity.Commodities))
	for _, shipmentCommodity := range entity.Commodities {
		if shipmentCommodity == nil {
			continue
		}

		r.normalizeCommodity(entity, shipmentCommodity)

		switch {
		case shipmentCommodity.ID.IsNil():
			shipmentCommodity.ID = pulid.MustNew("sc_")
			if err = r.insertCommodity(ctx, tx, shipmentCommodity); err != nil {
				return err
			}
		case existingCommodities[shipmentCommodity.ID] != nil:
			if err = r.updateCommodity(
				ctx,
				tx,
				shipmentCommodity,
				existingCommodities[shipmentCommodity.ID],
			); err != nil {
				return err
			}
			updatedCommodityIDs[shipmentCommodity.ID] = struct{}{}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown commodity").
				WithParam("shipmentCommodityId", shipmentCommodity.ID.String())
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingCommodities))
	for id := range existingCommodities {
		if _, ok := updatedCommodityIDs[id]; ok {
			continue
		}
		if _, ok := r.findCommodityInPayload(entity, id); ok {
			continue
		}
		deleteIDs = append(deleteIDs, id)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	_, err = tx.NewDelete().
		Model((*shipment.ShipmentCommodity)(nil)).
		Where("id IN (?)", bun.List(deleteIDs)).
		Where("shipment_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete shipment commodities: %w", err)
	}

	return nil
}

func (r *repository) getExistingCommodities(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.ShipmentCommodity, error) {
	commodities := make([]*shipment.ShipmentCommodity, 0)
	if err := tx.NewSelect().
		Model(&commodities).
		Where("shipment_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get existing shipment commodities: %w", err)
	}

	result := make(map[pulid.ID]*shipment.ShipmentCommodity, len(commodities))
	for _, commodity := range commodities {
		result[commodity.ID] = commodity
	}

	return result, nil
}

func (r *repository) normalizeCommodity(
	entity *shipment.Shipment,
	shipmentCommodity *shipment.ShipmentCommodity,
) {
	shipmentCommodity.ShipmentID = entity.ID
	shipmentCommodity.OrganizationID = entity.OrganizationID
	shipmentCommodity.BusinessUnitID = entity.BusinessUnitID
}

func (r *repository) insertCommodity(
	ctx context.Context,
	tx bun.IDB,
	shipmentCommodity *shipment.ShipmentCommodity,
) error {
	_, err := tx.NewInsert().Model(shipmentCommodity).Returning("*").Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipment commodity %s: %w", shipmentCommodity.ID, err)
	}

	return nil
}

func (r *repository) updateCommodity(
	ctx context.Context,
	tx bun.IDB,
	shipmentCommodity *shipment.ShipmentCommodity,
	existing *shipment.ShipmentCommodity,
) error {
	ov := existing.Version
	shipmentCommodity.Version = ov + 1
	shipmentCommodity.UpdatedAt = timeutils.NowUnix()

	cols := buncolgen.ShipmentCommodityColumns
	results, err := tx.NewUpdate().
		Model((*shipment.ShipmentCommodity)(nil)).
		Set(cols.CommodityID.Set(), shipmentCommodity.CommodityID).
		Set(cols.Weight.Set(), shipmentCommodity.Weight).
		Set(cols.Pieces.Set(), shipmentCommodity.Pieces).
		Set(cols.LengthFeet.Set(), shipmentCommodity.LengthFeet).
		Set(cols.WidthFeet.Set(), shipmentCommodity.WidthFeet).
		Set(cols.HeightFeet.Set(), shipmentCommodity.HeightFeet).
		Set(cols.Version.Set(), shipmentCommodity.Version).
		Set(cols.UpdatedAt.Set(), shipmentCommodity.UpdatedAt).
		Where(cols.ID.Eq(), shipmentCommodity.ID).
		Where(cols.ShipmentID.Eq(), shipmentCommodity.ShipmentID).
		Where(cols.OrganizationID.Eq(), shipmentCommodity.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), shipmentCommodity.BusinessUnitID).
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update shipment commodity %s: %w", shipmentCommodity.ID, err)
	}

	return dberror.CheckRowsAffected(results, "Shipment commodity", shipmentCommodity.ID.String())
}

func (r *repository) findCommodityInPayload(
	entity *shipment.Shipment,
	id pulid.ID,
) (*shipment.ShipmentCommodity, bool) {
	for _, shipmentCommodity := range entity.Commodities {
		if shipmentCommodity != nil && shipmentCommodity.ID == id {
			return shipmentCommodity, true
		}
	}

	return nil, false
}
