package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func delayThresholdMinutes(control *tenant.ShipmentControl) int16 {
	if control == nil || !control.AutoDelayShipments {
		return shipmentstate.DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return shipmentstate.ResolveDelayThresholdMinutes(0)
	}

	return shipmentstate.ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}

func autoCancelThresholdDays(control *tenant.ShipmentControl) int8 {
	if control.AutoCancelShipmentsThreshold == nil {
		return shipmentstate.ResolveAutoCancelThresholdDays(0)
	}

	return shipmentstate.ResolveAutoCancelThresholdDays(*control.AutoCancelShipmentsThreshold)
}

func (s *service) getShipmentControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.ShipmentControl, error) {
	return s.controlRepo.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
}

func duplicateBOLCheckRequest(entity *shipment.Shipment) *shipmentports.DuplicateBOLCheckRequest {
	req := &shipmentports.DuplicateBOLCheckRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		BOL: entity.BOL,
	}
	if entity.ID.IsNotNil() {
		req.ShipmentID = &entity.ID
	}

	return req
}
