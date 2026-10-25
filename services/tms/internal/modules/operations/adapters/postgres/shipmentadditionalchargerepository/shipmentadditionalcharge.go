package shipmentadditionalchargerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var chargeErrors = dberror.Mapping{
	Entity: "Shipment additional charge",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_additional_charges_accessorial_charge",
			Field:   "accessorialChargeId",
			Message: "Accessorial charge does not exist in your organization",
		},
		{
			Name:    "fk_additional_charges_shipment",
			Field:   "shipmentId",
			Message: "Shipment does not exist in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB                   *postgres.Connection
	Logger               *zap.Logger
	OccurrenceRepository detentionports.DetentionOccurrenceRepository
}

type repository struct {
	db             *postgres.Connection
	l              *zap.Logger
	occurrenceRepo detentionports.DetentionOccurrenceRepository
}

func New(p Params) shipmentports.ShipmentAdditionalChargeRepository {
	return &repository{
		db:             p.DB,
		l:              p.Logger.Named("postgres.shipment-additional-charge-repository"),
		occurrenceRepo: p.OccurrenceRepository,
	}
}

func (r *repository) SyncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	if entity.AdditionalCharges == nil {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.syncForShipment(ctx, entity)
	})
}

func (r *repository) syncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	existingCharges, err := r.getExistingCharges(ctx, entity)
	if err != nil {
		return err
	}

	updatedChargeIDs := make(map[pulid.ID]struct{}, len(entity.AdditionalCharges))
	inserts := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		normalizeCharge(entity, charge)

		switch {
		case charge.ID.IsNil():
			charge.ID = pulid.MustNew("ac_")
			inserts = append(inserts, charge)
		case existingCharges[charge.ID] != nil:
			if err = r.updateCharge(ctx, charge, existingCharges[charge.ID]); err != nil {
				return err
			}
			updatedChargeIDs[charge.ID] = struct{}{}
		default:
			return errortypes.NewBusinessError("Shipment contains an unknown additional charge").
				WithParam("additionalChargeId", charge.ID.String())
		}
	}

	if len(inserts) > 0 {
		if err = shipmentdb.AdditionalCharges.CreateMany(ctx, r.db.Keel(ctx), inserts); err != nil {
			return fmt.Errorf(
				"insert shipment additional charges: %w",
				chargeErrors.Map(err, inserts[0].ID.String()),
			)
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingCharges))
	deletedDetention := false
	for id, existing := range existingCharges {
		if _, ok := updatedChargeIDs[id]; ok {
			continue
		}
		if inPayload(entity, id) {
			continue
		}
		deleteIDs = append(deleteIDs, id)
		deletedDetention = deletedDetention || existing.Owner() == shipment.SystemOwnerDetention
	}

	if len(deleteIDs) > 0 {
		ac := shipmentdb.AdditionalCharges
		if _, err = ac.Where(ac.ID.In(deleteIDs...), ac.ShipmentID.Eq(entity.ID)).
			Delete(ctx, r.db.Keel(ctx)); err != nil {
			return fmt.Errorf("delete shipment additional charges: %w", err)
		}
	}

	if err = r.linkDetentionOccurrences(ctx, entity, deletedDetention); err != nil {
		return err
	}

	return r.reconcileHeaderTotals(ctx, entity)
}

func (r *repository) linkDetentionOccurrences(
	ctx context.Context,
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

	if err := r.occurrenceRepo.LinkCharges(ctx, &detentionports.LinkOccurrenceChargesRequest{
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

func (r *repository) reconcileHeaderTotals(ctx context.Context, entity *shipment.Shipment) error {
	freight := entity.FreightChargeAmount.Decimal
	totals, err := shipment.AdditionalChargeTotals(ctx, entity.AdditionalCharges, freight)
	if err != nil {
		return err
	}
	other := totals.Sum
	total := freight.Add(other)

	if entity.OtherChargeAmount.Valid && entity.OtherChargeAmount.Decimal.Equal(other) &&
		entity.TotalChargeAmount.Valid && entity.TotalChargeAmount.Decimal.Equal(total) {
		return nil
	}

	entity.OtherChargeAmount = decimal.NewNullDecimal(other)
	entity.TotalChargeAmount = decimal.NewNullDecimal(total)

	sp := shipmentdb.Shipments
	if _, err = sp.Where(sp.ID.Eq(entity.ID)).Update(
		ctx,
		r.db.Keel(ctx),
		sp.OtherChargeAmount.To(entity.OtherChargeAmount),
		sp.TotalChargeAmount.To(entity.TotalChargeAmount),
		sp.Version.ToExpr(sp.Version),
		sp.UpdatedAt.ToExpr(sp.UpdatedAt),
	); err != nil {
		return fmt.Errorf("reconcile shipment charge totals: %w", err)
	}

	return nil
}

func (r *repository) getExistingCharges(
	ctx context.Context,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.AdditionalCharge, error) {
	ac := shipmentdb.AdditionalCharges
	charges, err := ac.Where(ac.ShipmentID.Eq(entity.ID)).All(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("get existing shipment additional charges: %w", err)
	}

	result := make(map[pulid.ID]*shipment.AdditionalCharge, len(charges))
	for _, charge := range charges {
		result[charge.ID] = charge
	}

	return result, nil
}

func normalizeCharge(entity *shipment.Shipment, charge *shipment.AdditionalCharge) {
	charge.ShipmentID = entity.ID
	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
}

func (r *repository) updateCharge(
	ctx context.Context,
	charge *shipment.AdditionalCharge,
	existing *shipment.AdditionalCharge,
) error {
	ac := shipmentdb.AdditionalCharges
	n, err := ac.Where(
		ac.ID.Eq(charge.ID),
		ac.ShipmentID.Eq(charge.ShipmentID),
		ac.Version.Eq(existing.Version),
	).Update(
		ctx,
		r.db.Keel(ctx),
		ac.AccessorialChargeID.To(charge.AccessorialChargeID),
		ac.IsSystemGenerated.To(charge.IsSystemGenerated),
		ac.IsDetention.To(charge.IsDetention),
		ac.Method.To(charge.Method),
		ac.Amount.To(charge.Amount),
		ac.Unit.To(charge.Unit),
		ac.FuelSurchargeProgramID.ToExpr(keel.Val(charge.FuelSurchargeProgramID)),
		ac.FuelSurchargeDetail.To(charge.FuelSurchargeDetail),
		ac.RateAgreementAccessorialID.ToExpr(keel.Val(charge.RateAgreementAccessorialID)),
		ac.RateQuoteID.ToExpr(keel.Val(charge.RateQuoteID)),
	)
	if err != nil {
		return fmt.Errorf(
			"update shipment additional charge %s: %w",
			charge.ID,
			chargeErrors.Map(err, charge.ID.String()),
		)
	}

	if err = dberror.CheckAffected(
		n,
		"Shipment additional charge",
		charge.ID.String(),
	); err != nil {
		return err
	}

	charge.Version = existing.Version + 1
	charge.CreatedAt = existing.CreatedAt
	charge.UpdatedAt = timeutils.NowUnix()

	return nil
}

func inPayload(entity *shipment.Shipment, id pulid.ID) bool {
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.ID == id {
			return true
		}
	}

	return false
}
