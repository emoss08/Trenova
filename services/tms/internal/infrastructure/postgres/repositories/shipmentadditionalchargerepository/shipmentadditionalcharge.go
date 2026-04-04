package shipmentadditionalchargerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.ShipmentAdditionalChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-additional-charge-repository"),
	}
}

func (r *repository) SyncForShipment(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	existingCharges, err := r.getExistingCharges(ctx, tx, entity)
	if err != nil {
		return err
	}

	updatedChargeIDs := make(map[pulid.ID]struct{}, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		r.normalizeCharge(entity, charge)

		switch {
		case charge.ID.IsNil():
			charge.ID = pulid.MustNew("ac_")
			if err = r.insertCharge(ctx, tx, charge); err != nil {
				return err
			}
		case existingCharges[charge.ID] != nil:
			if err = r.updateCharge(ctx, tx, charge, existingCharges[charge.ID]); err != nil {
				return err
			}
			updatedChargeIDs[charge.ID] = struct{}{}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown additional charge").
				WithParam("additionalChargeId", charge.ID.String())
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingCharges))
	for id := range existingCharges {
		if _, ok := updatedChargeIDs[id]; ok {
			continue
		}
		if _, ok := r.findChargeInPayload(entity, id); ok {
			continue
		}
		deleteIDs = append(deleteIDs, id)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	_, err = tx.NewDelete().
		Model((*shipment.AdditionalCharge)(nil)).
		Where("id IN (?)", bun.List(deleteIDs)).
		Where("shipment_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete shipment additional charges: %w", err)
	}

	return nil
}

func (r *repository) getExistingCharges(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.AdditionalCharge, error) {
	charges := make([]*shipment.AdditionalCharge, 0)
	if err := tx.NewSelect().
		Model(&charges).
		Where("shipment_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get existing shipment additional charges: %w", err)
	}

	result := make(map[pulid.ID]*shipment.AdditionalCharge, len(charges))
	for _, charge := range charges {
		result[charge.ID] = charge
	}

	return result, nil
}

func (r *repository) normalizeCharge(entity *shipment.Shipment, charge *shipment.AdditionalCharge) {
	charge.ShipmentID = entity.ID
	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
}

func (r *repository) insertCharge(
	ctx context.Context,
	tx bun.IDB,
	charge *shipment.AdditionalCharge,
) error {
	_, err := tx.NewInsert().Model(charge).Returning("*").Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipment additional charge %s: %w", charge.ID, err)
	}

	return nil
}

func (r *repository) updateCharge(
	ctx context.Context,
	tx bun.IDB,
	charge *shipment.AdditionalCharge,
	existing *shipment.AdditionalCharge,
) error {
	ov := existing.Version
	charge.Version = ov + 1
	charge.UpdatedAt = timeutils.NowUnix()

	results, err := tx.NewUpdate().
		Model((*shipment.AdditionalCharge)(nil)).
		Set("accessorial_charge_id = ?", charge.AccessorialChargeID).
		Set("is_system_generated = ?", charge.IsSystemGenerated).
		Set("method = ?", charge.Method).
		Set("amount = ?", charge.Amount).
		Set("unit = ?", charge.Unit).
		Set("version = ?", charge.Version).
		Set("updated_at = ?", charge.UpdatedAt).
		Where("id = ?", charge.ID).
		Where("shipment_id = ?", charge.ShipmentID).
		Where("organization_id = ?", charge.OrganizationID).
		Where("business_unit_id = ?", charge.BusinessUnitID).
		Where("version = ?", ov).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update shipment additional charge %s: %w", charge.ID, err)
	}

	return dberror.CheckRowsAffected(results, "Shipment additional charge", charge.ID.String())
}

func (r *repository) findChargeInPayload(
	entity *shipment.Shipment,
	id pulid.ID,
) (*shipment.AdditionalCharge, bool) {
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.ID == id {
			return charge, true
		}
	}

	return nil, false
}
