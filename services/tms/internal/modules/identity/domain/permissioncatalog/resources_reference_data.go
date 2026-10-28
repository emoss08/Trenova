package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerReferenceDataResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceServiceType.String(),
		DisplayName:        "Service Type",
		Description:        "Service type definitions",
		Category:           "Shipments",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

}
