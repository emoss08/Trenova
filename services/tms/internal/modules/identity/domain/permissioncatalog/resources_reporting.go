package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerReportingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceReport.String(),
		DisplayName: "Report",
		Description: "Report generation and management",
		Category:    "Reports",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View reports and cancel your own report runs",
			},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Create reports"},
			{Operation: permission.OpUpdate, DisplayName: "Update", Description: "Modify reports"},
			{Operation: permission.OpDelete, DisplayName: "Delete", Description: "Delete reports"},
			{Operation: permission.OpExport, DisplayName: "Export", Description: "Export reports"},
		},
		DefaultSensitivity: permission.SensitivityInternal,
		Scopes: []ScopeSupport{
			{Scope: permission.DataScopeOwn, EnforcedBy: ScopeEnforcedByApplication},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDashboard.String(),
		DisplayName: "Dashboard",
		Description: "Dashboard configuration",
		Category:    "Reports",
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View dashboards"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create dashboards",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify dashboards",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})
}
