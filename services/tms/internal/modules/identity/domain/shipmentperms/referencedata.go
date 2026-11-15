package shipmentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

func referenceDataResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:           permission.ResourceServiceType.String(),
			Feature:            platformcatalog.FeatureCoreTMS,
			DisplayName:        "Service Type",
			Description:        "Service type definitions",
			Category:           "Shipments",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
		},
	}
}
