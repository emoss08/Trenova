package shipmentwrite

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
)

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
