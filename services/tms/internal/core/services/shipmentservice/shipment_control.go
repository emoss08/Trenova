package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
	return s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
}

func duplicateBOLCheckRequest(entity *shipment.Shipment) *repositories.DuplicateBOLCheckRequest {
	req := &repositories.DuplicateBOLCheckRequest{
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
