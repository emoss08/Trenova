package permission

func (r *Registry) registerReportingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceReport.String(),
		DisplayName: "Report",
		Description: "Report generation and management",
		Category:    "Reporting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View reports"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create reports"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify reports"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete reports"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export reports"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDashboard.String(),
		DisplayName: "Dashboard",
		Description: "Dashboard configuration",
		Category:    "Reporting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View dashboards"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create dashboards"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify dashboards"},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}
