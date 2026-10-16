package permission

func (r *Registry) registerReportingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceReport.String(),
		DisplayName: "Report",
		Description: "Report generation and management",
		Category:    "Reports",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View reports and cancel your own report runs",
			},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create reports"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify reports"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete reports"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export reports"},
		},
		DefaultSensitivity: SensitivityInternal,
		Scopes: []ScopeSupport{
			{Scope: DataScopeOwn, EnforcedBy: ScopeEnforcedByApplication},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDashboard.String(),
		DisplayName: "Dashboard",
		Description: "Dashboard configuration",
		Category:    "Reports",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View dashboards"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create dashboards"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify dashboards"},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}
