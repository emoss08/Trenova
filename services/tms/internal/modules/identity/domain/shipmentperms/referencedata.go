package shipmentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func referenceDataResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:           permission.ResourceServiceType.String(),
			DisplayName:        "Service Type",
			Description:        "Service type definitions",
			Category:           "Shipments",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
		},
	}
}
