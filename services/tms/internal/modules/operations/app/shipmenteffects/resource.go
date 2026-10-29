package shipmenteffects

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type shipmentTenantResource interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
	GetID() pulid.ID
}
