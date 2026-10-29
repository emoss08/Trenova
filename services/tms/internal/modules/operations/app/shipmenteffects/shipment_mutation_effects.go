package shipmenteffects

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
)

func TenantRefForShipment(entity shipmentTenantResource) shipmenteventservice.TenantRef {
	return shipmenteventservice.TenantRef{
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
	}
}
