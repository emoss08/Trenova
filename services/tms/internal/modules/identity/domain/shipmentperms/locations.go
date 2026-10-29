package shipmentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func locationResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:    permission.ResourceLocation.String(),
			DisplayName: "Location",
			Description: "Location management",
			Category:    "Dispatch",
			ReferenceView: []string{
				"code",
				"name",
				"status",
				"addressLine1",
				"addressLine2",
				"city",
				"stateId",
				"state",
				"postalCode",
				"latitude",
				"longitude",
				"locationCategoryId",
				"timezone",
			},
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View locations"},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create locations",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify locations",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export location data",
				},
				{
					Operation:   permission.OpImport,
					DisplayName: "Import",
					Description: "Import locations",
				},
				{
					Operation:   permission.OpArchive,
					DisplayName: "Archive",
					Description: "Archive locations",
				},
				{
					Operation:   permission.OpRestore,
					DisplayName: "Restore",
					Description: "Restore archived locations",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceLocationCategory.String(),
			DisplayName: "Location Category",
			Description: "Location category definitions",
			Category:    "Dispatch",
			ReferenceView: []string{
				"name",
				"type",
				"color",
				"facilityType",
				"requiresAppointment",
			},
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
		},
	}
}
