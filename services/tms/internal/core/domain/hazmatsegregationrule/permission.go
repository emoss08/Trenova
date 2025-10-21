package hazmatsegregationrule

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*hazmatSegregationRulePermission)(nil)

type hazmatSegregationRulePermission struct{}

func NewHazmatSegregationRulePermission() permissionregistry.PermissionAware {
	return &hazmatSegregationRulePermission{}
}

func (h *hazmatSegregationRulePermission) GetResourceName() string {
	return "hazmat_segregation_rule"
}

func (h *hazmatSegregationRulePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new hazmat segregation rules to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View hazmat segregation rule information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify hazmat segregation rule details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove hazmat segregation rules (requires approval due to safety regulations)",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export hazmat segregation rules data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import hazmat segregation rules from other sources",
		),
	}
}

func (h *hazmatSegregationRulePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"safety_officer": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport,
		"compliance":     permission.OpRead | permission.OpExport,
		"read_only":      permission.OpRead,
	}
}

func (h *hazmatSegregationRulePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *hazmatSegregationRulePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
