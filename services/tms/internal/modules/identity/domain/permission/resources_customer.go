package permission

func (r *Registry) registerCustomerResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCustomer.String(),
		DisplayName: "Customer",
		Description: "Customer management",
		Category:    "Customers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View customers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create customers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify customers"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export customer data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import customers"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive customers"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived customers",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFactoringCompany.String(),
		DisplayName: "Factoring Company",
		Description: "Factors that customers' invoices are assigned to",
		Category:    "Customers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View factoring companies"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add factoring companies"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify factoring companies"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export factoring companies"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})
}
