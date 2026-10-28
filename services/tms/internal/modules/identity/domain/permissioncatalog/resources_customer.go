package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerCustomerResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCustomer.String(),
		DisplayName: "Customer",
		Description: "Customer management",
		Category:    "Billing",
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
			"isGeocoded",
			"placeId",
			"externalId",
			"allowConsolidation",
			"exclusiveConsolidation",
			"consolidationPriority",
			"ediPartner",
		},
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View customers"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create customers",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify customers",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export customer data",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import customers",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive customers",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived customers",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceFactoringCompany.String(),
		DisplayName:   "Factoring Company",
		Description:   "Factors that customers' invoices are assigned to",
		Category:      "Billing",
		ReferenceView: []string{"code", "name"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View factoring companies",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add factoring companies",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify factoring companies",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export factoring companies",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})
}
