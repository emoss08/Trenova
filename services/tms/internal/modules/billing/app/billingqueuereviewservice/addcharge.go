package billingqueuereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) addChargeEffect(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	effect billingqueue.IssueEffect,
	actor *permissionports.RequestActor,
	tenantInfo pagination.TenantInfo,
) (*billingqueue.EffectSnapshot, error) {
	edit, err := planChargeAdd(item, effect)
	if err != nil {
		return nil, err
	}

	if _, err = s.queue.SettleCharges(ctx, &billingqueueports.UpdateChargesRequest{
		ItemID:            item.ID,
		AdditionalCharges: edit.charges,
		TenantInfo:        tenantInfo,
	}, actor); err != nil {
		return nil, err
	}

	addedID, err := s.newChargeID(ctx, item, edit.existing, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &billingqueue.EffectSnapshot{AddedChargeID: addedID}, nil
}

func (s *Service) removeAddedCharge(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	snapshot *billingqueue.EffectSnapshot,
	actor *permissionports.RequestActor,
	tenantInfo pagination.TenantInfo,
) error {
	if item.Shipment == nil {
		return errortypes.NewBusinessError("The shipment's charges could not be read")
	}

	kept := make([]*shipment.AdditionalCharge, 0, len(item.Shipment.AdditionalCharges))
	for _, charge := range item.Shipment.AdditionalCharges {
		if charge == nil || charge.ID == snapshot.AddedChargeID {
			continue
		}
		clone := *charge
		kept = append(kept, &clone)
	}

	_, err := s.queue.SettleCharges(ctx, &billingqueueports.UpdateChargesRequest{
		ItemID:            item.ID,
		AdditionalCharges: kept,
		TenantInfo:        tenantInfo,
	}, actor)

	return err
}

func (s *Service) newChargeID(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	existing map[pulid.ID]struct{},
	tenantInfo pagination.TenantInfo,
) (pulid.ID, error) {
	shp, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return pulid.Nil, err
	}
	for _, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		if _, ok := existing[charge.ID]; !ok {
			return charge.ID, nil
		}
	}

	return pulid.Nil, nil
}

func planChargeAdd(
	item *billingqueue.BillingQueueItem,
	effect billingqueue.IssueEffect,
) (*chargeEdit, error) {
	if item.Shipment == nil {
		return nil, errortypes.NewBusinessError("The shipment's charges could not be read")
	}
	if effect.AccessorialChargeID.IsNil() || !effect.Amount.Valid ||
		!effect.Amount.Decimal.IsPositive() {
		return nil, errortypes.NewBusinessError("This choice has no charge to add")
	}

	count := len(item.Shipment.AdditionalCharges)
	edit := &chargeEdit{
		charges:  make([]*shipment.AdditionalCharge, 0, count+1),
		existing: make(map[pulid.ID]struct{}, count),
	}
	for _, charge := range item.Shipment.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.AccessorialChargeID == effect.AccessorialChargeID {
			return nil, errortypes.NewValidationError(
				"issueId", errortypes.ErrInvalidOperation,
				"The shipment already bills this charge",
			)
		}
		edit.existing[charge.ID] = struct{}{}
		clone := *charge
		edit.charges = append(edit.charges, &clone)
	}

	added := &shipment.AdditionalCharge{
		OrganizationID:      item.OrganizationID,
		BusinessUnitID:      item.BusinessUnitID,
		ShipmentID:          item.ShipmentID,
		AccessorialChargeID: effect.AccessorialChargeID,
		Method:              accessorialcharge.MethodFlat,
		Amount:              effect.Amount.Decimal,
		Unit:                1,
	}
	edit.after = added
	edit.charges = append(edit.charges, added)

	return edit, nil
}

func addedCharge(
	item *billingqueue.BillingQueueItem,
	chargeID pulid.ID,
) *shipment.AdditionalCharge {
	if item.Shipment == nil {
		return nil
	}
	for _, charge := range item.Shipment.AdditionalCharges {
		if charge != nil && charge.ID == chargeID {
			return charge
		}
	}

	return nil
}
