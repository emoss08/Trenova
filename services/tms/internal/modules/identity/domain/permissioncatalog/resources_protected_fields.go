package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerProtectedFieldResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceProtectedPersonal.String(),
		DisplayName: "Personal details",
		Description: "A worker's date of birth, licence, TWIC and credential numbers, and drug test dates",
		Category:    "Protected fields",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "See personal details",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceProtectedMargins.String(),
		DisplayName: "Margins",
		Description: "Every margin figure: margins and margin percents, margin floors and targets, and a cost shown beside the revenue it gives the margin of",
		Category:    "Protected fields",
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "See margins"},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})
}
