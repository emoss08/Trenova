package shipmentadditionalchargerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                   *postgres.Connection
	Logger               *zap.Logger
	OccurrenceRepository repositories.DetentionOccurrenceRepository
}

type repository struct {
	db             *postgres.Connection
	l              *zap.Logger
	occurrenceRepo repositories.DetentionOccurrenceRepository
}

func New(p Params) repositories.ShipmentAdditionalChargeRepository {
	return &repository{
		db:             p.DB,
		l:              p.Logger.Named("postgres.shipment-additional-charge-repository"),
		occurrenceRepo: p.OccurrenceRepository,
	}
}

func (r *repository) SyncForShipment(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	if entity.AdditionalCharges == nil {
		return nil
	}

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
	deletedDetention := false
	for id, existing := range existingCharges {
		if _, ok := updatedChargeIDs[id]; ok {
			continue
		}
		if _, ok := r.findChargeInPayload(entity, id); ok {
			continue
		}
		deleteIDs = append(deleteIDs, id)
		deletedDetention = deletedDetention || existing.Owner() == shipment.SystemOwnerDetention
	}

	if len(deleteIDs) > 0 {
		if _, err = tx.NewDelete().
			Model((*shipment.AdditionalCharge)(nil)).
			Where("id IN (?)", bun.List(deleteIDs)).
			Where("shipment_id = ?", entity.ID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete shipment additional charges: %w", err)
		}
	}

	if err = r.linkDetentionOccurrences(ctx, tx, entity, deletedDetention); err != nil {
		return err
	}

	return r.reconcileHeaderTotals(ctx, tx, entity)
}

func (r *repository) linkDetentionOccurrences(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
	deletedDetention bool,
) error {
	if r.occurrenceRepo == nil {
		return nil
	}

	links := make(map[pulid.ID][]pulid.ID, 1)
	for _, charge := range entity.AdditionalCharges {
		if charge == nil || charge.Owner() != shipment.SystemOwnerDetention ||
			len(charge.DetentionOccurrenceIDs) == 0 {
			continue
		}
		links[charge.ID] = charge.DetentionOccurrenceIDs
	}

	if len(links) == 0 && !deletedDetention {
		return nil
	}

	if err := r.occurrenceRepo.LinkCharges(ctx, tx, &repositories.LinkOccurrenceChargesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ShipmentID:        entity.ID,
		ChargeOccurrences: links,
	}); err != nil {
		return fmt.Errorf("link detention occurrences: %w", err)
	}

	return nil
}

func (r *repository) reconcileHeaderTotals(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	freight := entity.FreightChargeAmount.Decimal
	other := shipment.AdditionalChargesTotal(entity.AdditionalCharges, freight)
	total := freight.Add(other)

	if entity.OtherChargeAmount.Valid && entity.OtherChargeAmount.Decimal.Equal(other) &&
		entity.TotalChargeAmount.Valid && entity.TotalChargeAmount.Decimal.Equal(total) {
		return nil
	}

	entity.OtherChargeAmount = decimal.NewNullDecimal(other)
	entity.TotalChargeAmount = decimal.NewNullDecimal(total)

	sp := buncolgen.ShipmentColumns
	if _, err := tx.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set(sp.OtherChargeAmount.Set(), other).
		Set(sp.TotalChargeAmount.Set(), total).
		Where(sp.ID.Eq(), entity.ID).
		Where(sp.OrganizationID.Eq(), entity.OrganizationID).
		Where(sp.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Exec(ctx); err != nil {
		return fmt.Errorf("reconcile shipment charge totals: %w", err)
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

	cols := buncolgen.AdditionalChargeColumns
	results, err := tx.NewUpdate().
		Model(charge).
		Column(
			cols.AccessorialChargeID.Bare(),
			cols.IsSystemGenerated.Bare(),
			cols.IsDetention.Bare(),
			cols.Method.Bare(),
			cols.Amount.Bare(),
			cols.Unit.Bare(),
			cols.FuelSurchargeProgramID.Bare(),
			cols.FuelSurchargeDetail.Bare(),
			cols.RateAgreementAccessorialID.Bare(),
			cols.RateQuoteID.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
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
