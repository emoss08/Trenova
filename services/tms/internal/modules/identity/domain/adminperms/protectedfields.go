package adminperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

func protectedFieldResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:    permission.ResourceProtectedPersonal.String(),
			Feature:     platformcatalog.FeatureAdministration,
			DisplayName: "Personal details",
			Description: "A worker's date of birth, licence, TWIC and credential numbers, and drug test dates",
			Category:    "Protected fields",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "See personal details",
				},
			},
			DefaultSensitivity: permission.SensitivityConfidential,
		},

		{
			Resource:    permission.ResourceProtectedMargins.String(),
			Feature:     platformcatalog.FeatureAdministration,
			DisplayName: "Margins",
			Description: "Every margin figure: margins and margin percents, margin floors and targets, and a cost shown beside the revenue it gives the margin of",
			Category:    "Protected fields",
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "See margins"},
			},
			DefaultSensitivity: permission.SensitivityConfidential,
		},
	}
}
