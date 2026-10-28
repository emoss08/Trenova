package reportperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func Resources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:    permission.ResourceReport.String(),
			DisplayName: "Report",
			Description: "Report generation and management",
			Category:    "Reports",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View reports and cancel your own report runs",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create reports",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify reports",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Delete reports",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export reports",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
			Scopes: []permissionspec.ScopeSupport{
				{
					Scope:      permission.DataScopeOwn,
					EnforcedBy: permissionspec.ScopeEnforcedByApplication,
				},
			},
		},

		{
			Resource:    permission.ResourceDashboard.String(),
			DisplayName: "Dashboard",
			Description: "Dashboard configuration",
			Category:    "Reports",
			Operations: []permissionspec.OperationDefinition{
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
		},
	}
}
